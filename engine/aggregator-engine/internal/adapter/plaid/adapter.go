package plaid

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"aggregator-engine/internal/adapter"
	"aggregator-engine/internal/domain"
	"aggregator-engine/internal/pkg/logger"

	"github.com/plaid/plaid-go/plaid"
)

var _ adapter.BrokerAdapter = (*PlaidAdapter)(nil)

type PlaidAdapter struct {
	client      *plaid.APIClient
	accessToken string
}

func NewPlaidAdapter(client *plaid.APIClient, accessToken string) *PlaidAdapter {
	return &PlaidAdapter{
		client:      client,
		accessToken: accessToken,
	}
}

func (a *PlaidAdapter) FetchAccountBalance(ctx context.Context, accountID string) (*domain.AccountBalance, error) {
	start := time.Now()
	logger.Log.Info("Fetching Plaid account balance", slog.String("account_id", accountID))

	request := plaid.NewAccountsBalanceGetRequest(a.accessToken)
	request.SetOptions(plaid.AccountsBalanceGetRequestOptions{
		AccountIds: &[]string{accountID},
	})

	resp, _, err := a.client.PlaidApi.AccountsBalanceGet(ctx).AccountsBalanceGetRequest(*request).Execute()
	duration := time.Since(start)

	if err != nil {
		errMsg := err.Error()
		if plaidErr, ok := err.(plaid.GenericOpenAPIError); ok {
			errMsg = string(plaidErr.Body())
		}

		logger.Log.Error("Plaid AccountsBalanceGet failed",
			slog.String("account_id", accountID),
			slog.String("error", errMsg),
			slog.Int64("duration_ms", duration.Milliseconds()),
		)
		return nil, fmt.Errorf("plaid balance get failed: %s", errMsg)
	}

	accounts := resp.GetAccounts()
	if len(accounts) == 0 {
		logger.Log.Warn("No accounts returned from Plaid balance query", slog.String("account_id", accountID))
		return nil, fmt.Errorf("no accounts found for id: %s", accountID)
	}

	acc := accounts[0]
	balances := acc.GetBalances()

	var totalCash float64
	if balances.Available.IsSet() && balances.Available.Get() != nil {
		totalCash = float64(*balances.Available.Get())
	} else if balances.Current.IsSet() && balances.Current.Get() != nil {
		totalCash = float64(*balances.Current.Get())
	}

	logger.Log.Info("Successfully fetched Plaid balance",
		slog.String("account_id", accountID),
		slog.Float64("total_cash", totalCash),
		slog.Int64("duration_ms", duration.Milliseconds()),
	)

	return &domain.AccountBalance{
		AccountID:   accountID,
		TotalEquity: totalCash,
		TotalCash:   totalCash,
		UpdatedAt:   time.Now().UTC(),
	}, nil
}

// FetchPositions satisfies the BrokerAdapter interface
func (a *PlaidAdapter) FetchPositions(ctx context.Context, accountID string) ([]domain.Position, error) {
	start := time.Now()
	logger.Log.Info("Fetching Plaid holdings and positions", slog.String("account_id", accountID))

	request := plaid.NewInvestmentsHoldingsGetRequest(a.accessToken)

	// Only apply the filter if a specific accountID was requested
	if accountID != "" {
		request.SetOptions(plaid.InvestmentHoldingsGetRequestOptions{
			AccountIds: &[]string{accountID},
		})
	}

	resp, _, err := a.client.PlaidApi.InvestmentsHoldingsGet(ctx).InvestmentsHoldingsGetRequest(*request).Execute()
	duration := time.Since(start)

	if err != nil {
		errMsg := err.Error()
		if plaidErr, ok := err.(plaid.GenericOpenAPIError); ok {
			errMsg = string(plaidErr.Body())
		}

		logger.Log.Error("Plaid API holdings request failed",
			slog.String("error", errMsg),
			slog.Int64("duration_ms", duration.Milliseconds()),
		)
		return nil, fmt.Errorf("plaid holdings request failed: %s", errMsg)
	}

	securitiesMap := make(map[string]plaid.Security)
	for _, sec := range resp.GetSecurities() {
		securitiesMap[sec.GetSecurityId()] = sec
	}

	var parsedPositions []domain.Position
	now := time.Now().UTC()

	for _, holding := range resp.GetHoldings() {
		sec, exists := securitiesMap[holding.GetSecurityId()]
		if !exists {
			continue
		}

		ticker := sec.GetTickerSymbol()
		if ticker == "" {
			ticker = sec.GetName()
		}

		pos := domain.Position{
			ID:           holding.GetSecurityId(),
			AccountID:    holding.GetAccountId(), // CRITICAL: We keep Plaid's true sub-account ID here
			Symbol:       ticker,
			Quantity:     float64(holding.GetQuantity()),
			CostBasis:    float64(holding.GetCostBasis()),
			MarketValue:  float64(holding.GetInstitutionValue()),
			LastSyncedAt: now,
		}

		pos.UnrealizedPL = pos.MarketValue - (pos.Quantity * pos.CostBasis)

		secType := sec.GetType()
		if secType == "equity" || secType == "etf" || secType == "mutual fund" {
			pos.AssetClass = domain.AssetClassEquity
		} else if secType == "derivative" {
			pos.AssetClass = domain.AssetClassOption
			if optData, err := domain.ParseOCC(ticker); err == nil {
				pos.OptionData = optData
			}
		} else {
			continue
		}

		parsedPositions = append(parsedPositions, pos)
	}

	return parsedPositions, nil
}

func (a *PlaidAdapter) FetchTransactions(ctx context.Context, accountID string, startDate, endDate time.Time) ([]domain.Transaction, error) {
	start := time.Now()
	logger.Log.Info("Fetching Plaid historical transactions",
		slog.String("account_id", accountID),
		slog.Time("start_date", startDate),
		slog.Time("end_date", endDate),
	)

	startStr := startDate.Format("2006-01-02")
	endStr := endDate.Format("2006-01-02")

	var parsedTransactions []domain.Transaction
	var offset int32 = 0
	const count int32 = 500

	for {
		txReq := plaid.NewInvestmentsTransactionsGetRequest(a.accessToken, startStr, endStr)
		options := plaid.InvestmentsTransactionsGetRequestOptions{}
		options.SetCount(count)
		options.SetOffset(offset)

		if accountID != "" {
			options.SetAccountIds([]string{accountID})
		}

		txReq.SetOptions(options)

		txResp, _, err := a.client.PlaidApi.InvestmentsTransactionsGet(ctx).InvestmentsTransactionsGetRequest(*txReq).Execute()

		if err != nil {
			errMsg := err.Error()
			if plaidErr, ok := err.(plaid.GenericOpenAPIError); ok {
				errMsg = string(plaidErr.Body())
			}

			logger.Log.Error("Plaid API transactions request failed",
				slog.String("account_id", accountID),
				slog.String("error", errMsg),
				slog.Int("offset", int(offset)),
			)
			return nil, fmt.Errorf("plaid transactions request failed: %s", errMsg)
		}

		txSecMap := make(map[string]string)
		for _, sec := range txResp.GetSecurities() {
			if sec.TickerSymbol.IsSet() {
				txSecMap[sec.GetSecurityId()] = sec.GetTickerSymbol()
			}
		}

		for _, t := range txResp.GetInvestmentTransactions() {
			symbol := "CASH"
			if t.SecurityId.IsSet() {
				if sym, ok := txSecMap[t.GetSecurityId()]; ok {
					symbol = sym
				}
			}

			parsedTransactions = append(parsedTransactions, domain.Transaction{
				ID:        t.GetInvestmentTransactionId(),
				AccountID: t.GetAccountId(),
				Symbol:    symbol,
				Date:      t.GetDate(),
				Name:      t.GetName(),
				Quantity:  float64(t.GetQuantity()),
				Price:     float64(t.GetPrice()),
				Amount:    float64(t.GetAmount()),
				Type:      t.GetType(),
				// FIXED: InstitutionName completely removed
			})
		}

		if len(txResp.GetInvestmentTransactions()) < int(count) {
			break
		}
		offset += count
	}

	logger.Log.Info("Successfully fetched Plaid transactions",
		slog.Int("count", len(parsedTransactions)),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
	)

	return parsedTransactions, nil
}
