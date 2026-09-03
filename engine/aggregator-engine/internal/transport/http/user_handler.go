package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"aggregator-engine/internal/repository"
)

type UserHandler struct {
	repo repository.UserRepo
}

func NewUserHandler(repo repository.UserRepo) *UserHandler {
	return &UserHandler{repo: repo}
}

type CreateUserRequest struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func (h *UserHandler) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ID == "" || req.Email == "" {
		http.Error(w, "ID and Email are required", http.StatusBadRequest)
		return
	}

	if err := h.repo.CreateUser(r.Context(), req.ID, req.Email); err != nil {
		slog.Error("Failed to create user in DB", slog.String("error", err.Error()), slog.String("user_id", req.ID))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	slog.Info("User successfully registered in engine", slog.String("user_id", req.ID))

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
