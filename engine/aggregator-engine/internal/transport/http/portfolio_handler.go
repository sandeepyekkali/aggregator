package http

import (
	"encoding/json"
	"net/http"

	"aggregator-engine/internal/service"
)

type PortfolioHandler struct {
	portfolioService *service.PortfolioService
}

func NewPortfolioHandler(svc *service.PortfolioService) *PortfolioHandler {
	return &PortfolioHandler{
		portfolioService: svc,
	}
}

// HandleGetPortfolio returns the aggregated portfolio for the authenticated user.
func (h *PortfolioHandler) HandleGetPortfolio(w http.ResponseWriter, r *http.Request) {
	// In a real application, userID comes from the JWT/Session context middleware.
	// For testing, we are mocking it via a request header
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized: missing user ID", http.StatusUnauthorized)
		return
	}

	// ... rest of your handler logic ...

	positions, err := h.portfolioService.GetAggregatedPortfolio(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error": "failed to fetch portfolio"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(positions); err != nil {
		http.Error(w, `{"error": "failed to encode response"}`, http.StatusInternalServerError)
	}
}
