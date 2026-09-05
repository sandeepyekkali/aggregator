package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"aggregator-engine/internal/pkg/logger"
	"aggregator-engine/internal/repository"
)

// SnapTradeHandler manages HTTP requests related to Premium brokerage connections.
type SnapTradeHandler struct {
	clientID      string
	consumerKey   string
	snapTradeRepo *repository.PostgresSnapTradeRepo
}

func NewSnapTradeHandler(clientID, consumerKey string, repo *repository.PostgresSnapTradeRepo) *SnapTradeHandler {
	return &SnapTradeHandler{
		clientID:      clientID,
		consumerKey:   consumerKey,
		snapTradeRepo: repo,
	}
}

// GenerateLink handles GET /api/v1/snaptrade/link
// It authenticates the user with SnapTrade and returns the secure Connection Portal URL.
func (h *SnapTradeHandler) GenerateLink(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract the user ID dynamically from the Auth middleware context
	// NOTE: If your internal/middleware/auth.go uses a custom type (e.g., middleware.UserIDKey),
	// update "user_id" to match that exact context key.
	userID, ok := ctx.Value("user_id").(string)
	if !ok || userID == "" {
		logger.Log.Error("Unauthorized request: missing user context in GenerateLink")
		http.Error(w, "Unauthorized: Invalid or missing user token", http.StatusUnauthorized)
		return
	}

	// Fetch the user's SnapTrade secret from the database
	secret, err := h.snapTradeRepo.GetSecret(ctx, userID)
	if err != nil || secret == "" {
		// If the user doesn't have a secret, they are new to the Premium tier.
		// We must register them with SnapTrade first to generate a userSecret.
		secret, err = h.registerSnapTradeUser(userID)
		if err != nil {
			logger.Log.Error("Failed to register new SnapTrade user", "error", err, "user_id", userID)
			http.Error(w, "Failed to initialize premium account", http.StatusInternalServerError)
			return
		}

		// Save the newly generated secret to the snaptrade_users table
		_ = h.snapTradeRepo.SaveSecret(ctx, userID, secret)
	}

	// Request the Connection Portal URL from SnapTrade
	loginURL := fmt.Sprintf("https://api.snaptrade.com/api/v1/authentication/login?clientId=%s&consumerKey=%s", h.clientID, h.consumerKey)

	reqBody, _ := json.Marshal(map[string]string{
		"userId":     userID,
		"userSecret": secret,
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", loginURL, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != 200 {
		logger.Log.Error("Failed to generate SnapTrade portal link", "status", res.StatusCode, "user_id", userID)
		http.Error(w, "Failed to generate connection link", http.StatusBadGateway)
		return
	}
	defer res.Body.Close()

	// Parse the redirect URI and return it to the React frontend
	var loginResp struct {
		RedirectURI string `json:"redirectURI"`
	}
	if err := json.NewDecoder(res.Body).Decode(&loginResp); err != nil {
		http.Error(w, "Invalid response from broker gateway", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"redirect_uri": loginResp.RedirectURI,
	})
}

// registerSnapTradeUser calls the SnapTrade API to provision a new user and returns their secret.
func (h *SnapTradeHandler) registerSnapTradeUser(userID string) (string, error) {
	url := fmt.Sprintf("https://api.snaptrade.com/api/v1/snapTrade/registerUser?clientId=%s&consumerKey=%s", h.clientID, h.consumerKey)

	reqBody, _ := json.Marshal(map[string]string{"userId": userID})
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != 200 {
		return "", fmt.Errorf("registration request failed with status: %d", res.StatusCode)
	}
	defer res.Body.Close()

	var regResp struct {
		UserSecret string `json:"userSecret"`
	}
	if err := json.NewDecoder(res.Body).Decode(&regResp); err != nil {
		return "", fmt.Errorf("failed to decode registration response")
	}

	return regResp.UserSecret, nil
}
