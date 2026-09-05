package http

import (
	"net/http"

	"aggregator-engine/internal/middleware"
)

// RouterConfig contains all handlers and middleware needed to construct the HTTP routes.
type RouterConfig struct {
	AuthMiddleware   func(http.Handler) http.Handler
	UserHandler      *UserHandler
	PortfolioHandler *PortfolioHandler
	TxHandler        *TransactionHandler
	PlaidHandler     *PlaidHandler
	SnapTradeHandler *SnapTradeHandler
}

// SetupRouter initializes all API endpoints, attaches middleware, and returns the root mux.
func SetupRouter(cfg RouterConfig) *http.ServeMux {
	mux := http.NewServeMux()

	// ==========================================
	// 1. Tier Middleware Helpers
	// ==========================================
	withTier := func(tier string, h http.HandlerFunc) http.Handler {
		return cfg.AuthMiddleware(middleware.RequireMinimumTier(tier)(h))
	}

	basic := func(h http.HandlerFunc) http.Handler { return withTier("basic", h) }
	// pro := func(h http.HandlerFunc) http.Handler { return withTier("pro", h) }
	premium := func(h http.HandlerFunc) http.Handler { return withTier("premium", h) }

	// ==========================================
	// 2. Public Routes (No Auth)
	// ==========================================
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "healthy"}`))
	})

	mux.HandleFunc("POST /api/v1/users", cfg.UserHandler.HandleCreateUser)

	// ==========================================
	// 3. Basic Tier (Available to Basic, Pro & Premium)
	// ==========================================
	// Core Data Endpoints - Updated to match the new portfolio_handler.go methods
	mux.Handle("GET /api/v1/portfolio", basic(cfg.PortfolioHandler.GetPortfolio))
	mux.Handle("GET /api/v1/connections", basic(cfg.PortfolioHandler.GetConnections))

	// Assuming txHandler retains its original method name
	mux.Handle("GET /api/v1/transactions", basic(cfg.TxHandler.HandleGetTransactions))

	// Plaid Integration - Retaining original method names
	mux.Handle("POST /api/v1/plaid/create-link-token", basic(cfg.PlaidHandler.HandleCreateLinkToken))
	mux.Handle("POST /api/v1/plaid/exchange-public-token", basic(cfg.PlaidHandler.HandleExchangePublicToken))

	// ==========================================
	// 4. Premium Tier (Strictly Gated to Premium)
	// ==========================================
	// Updated to match the new snaptrade_handler.go GenerateLink method
	mux.Handle("GET /api/v1/snaptrade/link", premium(cfg.SnapTradeHandler.GenerateLink))
	mux.Handle("POST /api/v1/snaptrade/link", premium(cfg.SnapTradeHandler.GenerateLink))

	return mux
}
