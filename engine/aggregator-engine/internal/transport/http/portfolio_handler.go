package http

import (
	"encoding/json"
	"net/http"

	"aggregator-engine/internal/pkg/logger"
	"aggregator-engine/internal/repository"
)

// PortfolioHandler manages HTTP requests related to user holdings and brokerage connections.
type PortfolioHandler struct {
	positionRepo *repository.PostgresPositionRepo
	userRepo     *repository.PostgresUserRepo // Repo containing GetConnectionSummaries
}

func NewPortfolioHandler(positionRepo *repository.PostgresPositionRepo, userRepo *repository.PostgresUserRepo) *PortfolioHandler {
	return &PortfolioHandler{
		positionRepo: positionRepo,
		userRepo:     userRepo,
	}
}

// GetPortfolio handles GET /api/v1/portfolio
// It retrieves all synced positions for the authenticated user across all connected brokerages.
func (h *PortfolioHandler) GetPortfolio(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract the user ID dynamically from the Auth middleware context
	// NOTE: Update "user_id" if your middleware uses a custom typed key (e.g., middleware.UserIDKey)
	userID, ok := ctx.Value("user_id").(string)
	if !ok || userID == "" {
		logger.Log.Error("Unauthorized request: missing user context in GetPortfolio")
		http.Error(w, "Unauthorized: Invalid or missing user token", http.StatusUnauthorized)
		return
	}

	positions, err := h.positionRepo.GetPositionsByUser(ctx, userID)
	if err != nil {
		logger.Log.Error("Failed to fetch portfolio positions", "error", err, "user_id", userID)
		http.Error(w, "Failed to load portfolio", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(positions)
}

// GetConnections handles GET /api/v1/connections
// It returns a grouped summary of all active and inactive broker integrations for the Manage Connections UI.
func (h *PortfolioHandler) GetConnections(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract the user ID dynamically from the Auth middleware context
	userID, ok := ctx.Value("user_id").(string)
	if !ok || userID == "" {
		logger.Log.Error("Unauthorized request: missing user context in GetConnections")
		http.Error(w, "Unauthorized: Invalid or missing user token", http.StatusUnauthorized)
		return
	}

	// Fetch the aggregated connection summaries mapped from the broker_accounts table
	connections, err := h.userRepo.GetConnectionSummaries(ctx, userID)
	if err != nil {
		logger.Log.Error("Failed to fetch connection summaries", "error", err, "user_id", userID)
		http.Error(w, "Failed to load connections", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(connections)
}
