package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"aggregator-engine/internal/repository"
	"aggregator-engine/internal/snaptrade"
)

type SnapTradeHandler struct {
	client *snaptrade.Client
	repo   repository.SnapTradeRepo // Using the isolated repository
}

func NewSnapTradeHandler(client *snaptrade.Client, repo repository.SnapTradeRepo) *SnapTradeHandler {
	return &SnapTradeHandler{
		client: client,
		repo:   repo,
	}
}

func (h *SnapTradeHandler) HandleCreateLinkToken(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 1. Check our isolated table for an existing secret
	secret, err := h.repo.GetSecret(r.Context(), userID)
	if err != nil {
		slog.Error("Failed to fetch SnapTrade secret from DB", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 2. If missing, register and encrypt/save
	if secret == "" {
		secret, err = h.client.RegisterUser(r.Context(), userID)
		if err != nil {
			slog.Error("SnapTrade registration failed", slog.String("error", err.Error()))
			http.Error(w, "Failed to register broker account", http.StatusBadGateway)
			return
		}

		if err := h.repo.SetSecret(r.Context(), userID, secret); err != nil {
			slog.Error("Failed to save encrypted SnapTrade secret", slog.String("error", err.Error()))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	// 3. Generate the UI login link
	redirectURI, err := h.client.GenerateLoginLink(r.Context(), userID, secret)
	if err != nil {
		slog.Error("SnapTrade link generation failed", slog.String("error", err.Error()))
		http.Error(w, "Failed to generate link", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"redirect_uri": redirectURI,
	})
}
