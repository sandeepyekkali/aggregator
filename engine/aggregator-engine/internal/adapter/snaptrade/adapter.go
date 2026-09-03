package snaptrade

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"aggregator-engine/internal/adapter"
	"aggregator-engine/internal/domain"
	"aggregator-engine/internal/pkg/logger"
)

var _ adapter.BrokerAdapter = (*SnapTradeAdapter)(nil)

type SnapTradeAdapter struct {
	clientID    string
	consumerKey string
	userID      string
	userSecret  string
	baseURL     string
	httpClient  *http.Client
}

func NewSnapTradeAdapter(clientID, consumerKey, userID, userSecret string) *SnapTradeAdapter {
	return &SnapTradeAdapter{
		clientID:    clientID,
		consumerKey: consumerKey,
		userID:      userID,
		userSecret:  userSecret,
		baseURL:     "https://api.snaptrade.com/api/v1",
		httpClient:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (a *SnapTradeAdapter) FetchAccountBalance(ctx context.Context, accountID string) (*domain.AccountBalance, error) {
	start := time.Now()
	logger.Log.Info("Fetching SnapTrade account balance", slog.String("account_id", accountID))

	url := fmt.Sprintf("%s/accounts/%s/balances?clientId=%s&consumerKey=%s&userId=%s&userSecret=%s",
		a.baseURL, accountID, a.clientID, a.consumerKey, a.userID, a.userSecret)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	res, err := a.httpClient.Do(req)
	duration := time.Since(start)

	if err != nil || res.StatusCode != 200 {
		logger.Log.Error("SnapTrade balances request failed",
			slog.String("account_id", accountID),
			slog.Int("status", res.StatusCode),
			slog.Int64("duration_ms", duration.Milliseconds()),
		)
		return nil, fmt.Errorf("snaptrade balance get failed: %v", err)
	}
	defer res.Body.Close()

	var balances struct {
		Total struct {
			Amount float64 `json:"amount"`
		} `json:"total"`
		TotalCash struct {
			Amount float64 `json:"amount"`
		} `json:"total_cash"`
	}

	if err := json.NewDecoder(res.Body).Decode(&balances); err != nil {
		return nil, err
	}

	return &domain.AccountBalance{
		AccountID:   accountID,
		TotalEquity: balances.Total.Amount,
		TotalCash:   balances.TotalCash.Amount,
		UpdatedAt:   time.Now().UTC(),
	}, nil
}

func (a *SnapTradeAdapter) FetchPositions(ctx context.Context, accountID string) ([]domain.Position, error) {
	start := time.Now()
	logger.Log.Info("Fetching SnapTrade real-time positions", slog.String("account_id", accountID))

	url := fmt.Sprintf("%s/accounts/%s/positions?clientId=%s&consumerKey=%s&userId=%s&userSecret=%s",
		a.baseURL, accountID, a.clientID, a.consumerKey, a.userID, a.userSecret)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	res, err := a.httpClient.Do(req)
	duration := time.Since(start)

	if err != nil || res.StatusCode != 200 {
		logger.Log.Error("SnapTrade positions request failed",
			slog.String("account_id", accountID),
			slog.Int64("duration_ms", duration.Milliseconds()),
		)
		return nil, fmt.Errorf("snaptrade positions request failed: %v", err)
	}
	defer res.Body.Close()

	var stPositions []struct {
		Symbol struct {
			Symbol string `json:"symbol"`
		} `json:"symbol"`
		OptionSymbol struct {
			Ticker         string  `json:"ticker"`
			ExpirationDate string  `json:"expiration_date"` // SnapTrade format: YYYY-MM-DD
			StrikePrice    float64 `json:"strike_price"`
			OptionType     string  `json:"option_type"` // CALL or PUT
		} `json:"option_symbol"`
		Price                float64 `json:"price"`
		Units                float64 `json:"units"`
		AveragePurchasePrice float64 `json:"average_purchase_price"`
	}

	if err := json.NewDecoder(res.Body).Decode(&stPositions); err != nil {
		return nil, err
	}

	var parsedPositions []domain.Position
	now := time.Now().UTC()

	for _, p := range stPositions {
		// Use the full symbol string provided by SnapTrade as the unique position ID
		uniqueID := p.Symbol.Symbol

		pos := domain.Position{
			ID:           uniqueID, // FIXED: Populated to prevent Postgres PK violations
			AccountID:    accountID,
			Quantity:     p.Units,
			CostBasis:    p.AveragePurchasePrice,
			MarketValue:  p.Price * p.Units,
			LastSyncedAt: now,
		}

		pos.UnrealizedPL = pos.MarketValue - (pos.Quantity * pos.CostBasis)

		if p.OptionSymbol.Ticker != "" {
			pos.AssetClass = domain.AssetClassOption
			pos.Symbol = p.Symbol.Symbol // Contains the full string (e.g., OCC) from SnapTrade

			expDate, err := time.Parse("2006-01-02", p.OptionSymbol.ExpirationDate)
			if err != nil {
				logger.Log.Warn("Failed to parse SnapTrade expiration date", slog.String("date", p.OptionSymbol.ExpirationDate))
			}

			optType := domain.OptionTypeCall
			if strings.ToUpper(p.OptionSymbol.OptionType) == "PUT" {
				optType = domain.OptionTypePut
			}

			pos.OptionData = &domain.OptionDetails{
				UnderlyingSymbol: p.OptionSymbol.Ticker,
				ExpirationDate:   expDate,
				OptionType:       optType,
				StrikePrice:      p.OptionSymbol.StrikePrice,
			}
		} else {
			pos.AssetClass = domain.AssetClassEquity
			pos.Symbol = p.Symbol.Symbol
		}

		parsedPositions = append(parsedPositions, pos)
	}

	return parsedPositions, nil
}

func (a *SnapTradeAdapter) FetchTransactions(ctx context.Context, accountID string, startDate, endDate time.Time) ([]domain.Transaction, error) {
	start := time.Now()
	logger.Log.Info("Fetching SnapTrade historical transactions", slog.String("account_id", accountID))

	startStr := startDate.Format("2006-01-02")
	endStr := endDate.Format("2006-01-02")

	url := fmt.Sprintf("%s/accounts/%s/activities?startDate=%s&endDate=%s&clientId=%s&consumerKey=%s&userId=%s&userSecret=%s",
		a.baseURL, accountID, startStr, endStr, a.clientID, a.consumerKey, a.userID, a.userSecret)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	res, err := a.httpClient.Do(req)
	duration := time.Since(start)

	if err != nil || res.StatusCode != 200 {
		logger.Log.Error("SnapTrade activities request failed",
			slog.String("account_id", accountID),
			slog.Int("status", res.StatusCode),
			slog.Int64("duration_ms", duration.Milliseconds()),
		)
		return nil, fmt.Errorf("snaptrade activities request failed: %v", err)
	}
	defer res.Body.Close()

	var activities []struct {
		ID          string  `json:"id"`
		TradeDate   string  `json:"trade_date"`
		Description string  `json:"description"`
		Type        string  `json:"type"`
		Amount      float64 `json:"amount"`
		Units       float64 `json:"units"`
		Price       float64 `json:"price"`

		Symbol struct {
			Symbol string `json:"symbol"`
		} `json:"symbol"`

		OptionSymbol struct {
			Ticker string `json:"ticker"`
		} `json:"option_symbol"`
	}

	if err := json.NewDecoder(res.Body).Decode(&activities); err != nil {
		return nil, err
	}

	var parsedTransactions []domain.Transaction
	for _, act := range activities {
		formattedDate := ""
		if len(act.TradeDate) >= 10 {
			formattedDate = act.TradeDate[:10]
		}

		var ticker string
		if act.OptionSymbol.Ticker != "" {
			ticker = act.OptionSymbol.Ticker
		} else {
			ticker = act.Symbol.Symbol
		}

		tx := domain.Transaction{
			ID:        act.ID,
			UserID:    a.userID,
			AccountID: accountID,
			Symbol:    ticker,
			Date:      formattedDate,
			Name:      act.Description,
			Quantity:  act.Units,
			Price:     act.Price,
			Amount:    act.Amount,
			Type:      act.Type,
			// FIXED: Removed InstitutionName entirely since it does not exist in the DB schema
		}

		parsedTransactions = append(parsedTransactions, tx)
	}

	logger.Log.Info("Successfully parsed SnapTrade transactions",
		slog.Int("count", len(parsedTransactions)),
		slog.Int64("duration_ms", duration.Milliseconds()),
	)

	return parsedTransactions, nil
}
