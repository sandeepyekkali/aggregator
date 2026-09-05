package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/plaid/plaid-go/plaid"

	plaidAdapter "aggregator-engine/internal/adapter/plaid"
	snaptradeAdapter "aggregator-engine/internal/adapter/snaptrade"
	"aggregator-engine/internal/config"
	"aggregator-engine/internal/domain"
	"aggregator-engine/internal/pkg/crypto"
	"aggregator-engine/internal/pkg/logger"
	"aggregator-engine/internal/repository"
)

type SnapTradeAccount struct {
	ID                     string `json:"id"`
	BrokerageAuthorization struct {
		Brokerage struct {
			Name string `json:"name"`
		} `json:"brokerage"`
	} `json:"brokerage_authorization"`
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Configuration failure", slog.String("error", err.Error()))
		os.Exit(1)
	}

	log := logger.InitLogger(cfg.AppEnv)
	log.Info("Starting Background Sync Worker (State & History)")

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Error("DB connection failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	plaidConfig := plaid.NewConfiguration()
	plaidConfig.AddDefaultHeader("PLAID-CLIENT-ID", cfg.PlaidClientID)
	plaidConfig.AddDefaultHeader("PLAID-SECRET", cfg.PlaidSecret)
	plaidConfig.UseEnvironment(plaid.Sandbox)
	plaidClient := plaid.NewAPIClient(plaidConfig)

	// In production, this must be sourced from an environment variable (e.g., cfg.EncryptionKey)
	testKey := []byte("0123456789abcdef0123456789abcdef")
	encryptor, err := crypto.NewEncryptor(testKey)
	if err != nil {
		log.Error("Failed to initialize encryptor", slog.String("error", err.Error()))
		os.Exit(1)
	}

	plaidRepo := repository.NewPostgresPlaidRepo(db, encryptor)
	snapTradeRepo := repository.NewPostgresSnapTradeRepo(db, encryptor)
	positionRepo := repository.NewPostgresPositionRepo(db)
	txRepo := repository.NewPostgresTransactionRepo(db)

	ctx := context.Background()

	log.Info("Discovering active users for background sync...")

	// 1. Fetch all unique users who have at least one active brokerage connection
	rows, err := db.QueryContext(ctx, `SELECT DISTINCT user_id FROM broker_accounts WHERE is_active = true`)
	if err != nil {
		log.Error("Failed to fetch active users from database", slog.String("error", err.Error()))
		os.Exit(1)
	}

	var activeUsers []string
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err == nil {
			activeUsers = append(activeUsers, uid)
		}
	}
	// Safely close the rows to free the database connection pool before the heavy loop operations start
	rows.Close()

	// 2. Iterate through every active user and execute their specific sync pipelines
	for _, userID := range activeUsers {
		log.Info("Starting sync for user", slog.String("user_id", userID))

		// ==========================================
		// 1. PLAID SYNC (BASIC TIER)
		// ==========================================
		if items, err := plaidRepo.GetItemsForUser(ctx, userID); err == nil {
			for _, item := range items {
				pAdapter := plaidAdapter.NewPlaidAdapter(plaidClient, item.AccessToken)

				positions, err := pAdapter.FetchPositions(ctx, "")
				if err != nil {
					log.Error("Failed to fetch Plaid positions", slog.String("error", err.Error()), slog.String("item_id", item.ItemID))
					continue
				}

				positionsByAccount := make(map[string][]domain.Position)
				for _, pos := range positions {
					pos.UserID = userID
					pos.Broker = domain.BrokerProvider("PLAID")
					positionsByAccount[pos.AccountID] = append(positionsByAccount[pos.AccountID], pos)
				}

				for accID, accountPositions := range positionsByAccount {
					_, err = db.ExecContext(ctx, `
						INSERT INTO broker_accounts (user_id, broker, account_id, item_id, institution_name)
						VALUES ($1, $2, $3, $4, $5)
						ON CONFLICT (user_id, broker, account_id) DO UPDATE 
						SET is_active = true, item_id = EXCLUDED.item_id, institution_name = EXCLUDED.institution_name;
					`, userID, "PLAID", accID, item.ItemID, item.InstitutionName)
					if err != nil {
						log.Error("Failed to upsert plaid broker account", slog.String("error", err.Error()), slog.String("account_id", accID))
						continue
					}

					_ = positionRepo.SyncAccountPositions(ctx, userID, accID, accountPositions)

					startDateObj := getIncrementalStartDate(ctx, txRepo, userID)
					endDateObj := time.Now()

					if txs, err := pAdapter.FetchTransactions(ctx, accID, startDateObj, endDateObj); err == nil {
						for i := range txs {
							txs[i].UserID = userID
							txs[i].Broker = "PLAID"
						}
						_ = txRepo.SyncTransactionWindow(ctx, userID, "PLAID", accID, startDateObj.Format("2006-01-02"), endDateObj.Format("2006-01-02"), txs)
					}
				}
			}
		}

		// ==========================================
		// 2. SNAPTRADE SYNC (PREMIUM TIER)
		// ==========================================
		if secret, err := snapTradeRepo.GetSecret(ctx, userID); err == nil && secret != "" {
			stAdapter := snaptradeAdapter.NewSnapTradeAdapter(cfg.SnapTradeClientID, cfg.SnapTradeConsumerKey, userID, secret)

			accountList := discoverSnapTradeAccounts(cfg.SnapTradeClientID, cfg.SnapTradeConsumerKey, userID, secret)

			for _, acc := range accountList {
				instName := acc.BrokerageAuthorization.Brokerage.Name

				_, err = db.ExecContext(ctx, `
					INSERT INTO broker_accounts (user_id, broker, account_id, institution_name)
					VALUES ($1, $2, $3, $4)
					ON CONFLICT (user_id, broker, account_id) DO UPDATE 
					SET is_active = true, institution_name = EXCLUDED.institution_name;
				`, userID, "SNAPTRADE", acc.ID, instName)
				if err != nil {
					log.Error("Failed to upsert snaptrade broker account", slog.String("error", err.Error()), slog.String("account_id", acc.ID))
					continue
				}

				if positions, err := stAdapter.FetchPositions(ctx, acc.ID); err == nil {
					for i := range positions {
						positions[i].UserID = userID
						positions[i].Broker = domain.BrokerProvider("SNAPTRADE")
					}
					_ = positionRepo.SyncAccountPositions(ctx, userID, acc.ID, positions)
				}

				startDateObj := getIncrementalStartDate(ctx, txRepo, userID)
				endDateObj := time.Now()

				if txs, err := stAdapter.FetchTransactions(ctx, acc.ID, startDateObj, endDateObj); err == nil {
					for i := range txs {
						txs[i].UserID = userID
						txs[i].Broker = "SNAPTRADE"
					}
					_ = txRepo.SyncTransactionWindow(ctx, userID, "SNAPTRADE", acc.ID, startDateObj.Format("2006-01-02"), endDateObj.Format("2006-01-02"), txs)
				}
			}
		}

		log.Info("Finished sync for user", slog.String("user_id", userID))
	}

	log.Info("Sync Worker Finished Successfully. Processed all active users.")
}

func getIncrementalStartDate(ctx context.Context, txRepo *repository.PostgresTransactionRepo, userID string) time.Time {
	startDateObj := time.Now().AddDate(-2, 0, 0)
	if latestDateStr, _ := txRepo.GetLatestTransactionDate(ctx, userID); latestDateStr != "" {
		if parsedDate, err := time.Parse("2006-01-02", latestDateStr); err == nil {
			startDateObj = parsedDate.AddDate(0, 0, -14)
		}
	}
	return startDateObj
}

func discoverSnapTradeAccounts(clientID, consumerKey, userID, secret string) []SnapTradeAccount {
	url := fmt.Sprintf("https://api.snaptrade.com/api/v1/accounts?clientId=%s&consumerKey=%s&userId=%s&userSecret=%s",
		clientID, consumerKey, userID, secret)

	req, _ := http.NewRequest("GET", url, nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != 200 {
		return nil
	}
	defer res.Body.Close()

	var accounts []SnapTradeAccount
	if err := json.NewDecoder(res.Body).Decode(&accounts); err != nil {
		return nil
	}

	return accounts
}
