package http

import (
	"aggregator-engine/internal/repository"
	"encoding/json"
	"net/http"
)

type TransactionHandler struct {
	txRepo *repository.PostgresTransactionRepo
}

func NewTransactionHandler(txRepo *repository.PostgresTransactionRepo) *TransactionHandler {
	return &TransactionHandler{txRepo: txRepo}
}

func (h *TransactionHandler) HandleGetTransactions(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized: missing user ID", http.StatusUnauthorized)
		return
	}

	txs, err := h.txRepo.GetUserTransactions(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to fetch transactions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(txs)
}
