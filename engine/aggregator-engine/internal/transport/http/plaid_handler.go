package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/plaid/plaid-go/plaid"

	"aggregator-engine/internal/pkg/logger"
	"aggregator-engine/internal/repository"
)

type PlaidHandler struct {
	client    *plaid.APIClient
	plaidRepo repository.PlaidRepository
}

func NewPlaidHandler(client *plaid.APIClient, plaidRepo repository.PlaidRepository) *PlaidHandler {
	return &PlaidHandler{
		client:    client,
		plaidRepo: plaidRepo,
	}
}

// HandleCreateLinkToken initializes the Plaid UI widget securely
func (h *PlaidHandler) HandleCreateLinkToken(w http.ResponseWriter, r *http.Request) {
	// 1. Authenticate Request
	// Safely extract the verified User ID from the context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized: missing user ID in context", http.StatusUnauthorized)
		return
	}

	// 2. Configure Link Token
	user := plaid.LinkTokenCreateRequestUser{
		ClientUserId: userID,
	}

	request := plaid.NewLinkTokenCreateRequest(
		"Portfolio Aggregator",
		"en",
		[]plaid.CountryCode{plaid.COUNTRYCODE_US},
		user,
	)

	// CRITICAL: We strictly request the Investments product so the token has permission to read holdings
	request.SetProducts([]plaid.Products{plaid.PRODUCTS_INVESTMENTS})

	// 3. Execute Plaid API Call
	resp, _, err := h.client.PlaidApi.LinkTokenCreate(r.Context()).LinkTokenCreateRequest(*request).Execute()
	if err != nil {
		errMsg := err.Error()
		if plaidErr, ok := err.(plaid.GenericOpenAPIError); ok {
			errMsg = string(plaidErr.Body())
		}

		logger.Log.Error("Plaid LinkTokenCreate failed",
			slog.String("user_id", userID),
			slog.String("error", errMsg),
		)

		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	logger.Log.Info("Successfully created Link Token", slog.String("user_id", userID))

	// 4. Return the temporary Link Token to the React frontend
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"link_token": resp.GetLinkToken()})
}

// HandleExchangePublicToken trades the temporary frontend token for a permanent access token
func (h *PlaidHandler) HandleExchangePublicToken(w http.ResponseWriter, r *http.Request) {
	// 1. Authenticate Request
	// Safely extract the verified User ID from the context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized: missing user ID in context", http.StatusUnauthorized)
		return
	}

	// 2. Parse Dynamic Frontend Payload
	var reqBody struct {
		PublicToken     string `json:"public_token"`
		InstitutionID   string `json:"institution_id"`
		InstitutionName string `json:"institution_name"` // <-- UPDATED: Capturing bank name from React
	}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		logger.Log.Warn("Invalid request body for exchange token")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if reqBody.PublicToken == "" {
		logger.Log.Warn("Empty public token submitted")
		http.Error(w, "Empty public token", http.StatusBadRequest)
		return
	}

	// 3. Execute Token Exchange
	request := plaid.NewItemPublicTokenExchangeRequest(reqBody.PublicToken)

	resp, _, err := h.client.PlaidApi.ItemPublicTokenExchange(r.Context()).ItemPublicTokenExchangeRequest(*request).Execute()
	if err != nil {
		errMsg := err.Error()
		if plaidErr, ok := err.(plaid.GenericOpenAPIError); ok {
			errMsg = string(plaidErr.Body())
		}

		logger.Log.Error("Plaid ItemPublicTokenExchange failed",
			slog.String("user_id", userID),
			slog.String("error", errMsg),
		)
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	accessToken := resp.GetAccessToken()
	itemID := resp.GetItemId()

	institutionName := reqBody.InstitutionName
	if institutionName == "" {
		institutionName = "Connected Broker"
	}

	// 4. Lock to PostgreSQL
	// We dynamically pass the user's selected brokerage ID and Name into the database constraint
	err = h.plaidRepo.SaveItem(r.Context(), repository.PlaidItem{
		ItemID:          itemID,
		UserID:          userID,
		AccessToken:     accessToken,
		InstitutionID:   reqBody.InstitutionID,
		InstitutionName: institutionName, // <-- UPDATED: Saving the real bank name
	})

	if err != nil {
		logger.Log.Error("Failed to save Plaid item to DB",
			slog.String("item_id", itemID),
			slog.String("user_id", userID),
			slog.String("error", err.Error()),
		)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	logger.Log.Info("Successfully exchanged and saved Plaid token",
		slog.String("user_id", userID),
		slog.String("item_id", itemID),
		slog.String("broker_name", institutionName), // <-- UPDATED log
	)

	// 5. Success
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "success"}`))
}
