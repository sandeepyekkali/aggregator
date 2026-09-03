package http

import (
	"net/http"
)

// SetupRouter initializes all API endpoints and attaches middleware.
func SetupRouter(portfolioHandler *PortfolioHandler) *http.ServeMux {
	mux := http.NewServeMux()

	// Health check for load balancers
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy"}`))
	})

	// Portfolio routes
	mux.HandleFunc("GET /api/v1/portfolio", portfolioHandler.HandleGetPortfolio)

	// Additional routes (Auth, Strategies, Trades) would be mounted here

	return mux
}
