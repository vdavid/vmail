package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vdavid/vmail/backend/internal/auth"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/models"
)

type AuthHandler struct {
	pool *pgxpool.Pool
}

func NewAuthHandler(pool *pgxpool.Pool) *AuthHandler {
	return &AuthHandler{pool: pool}
}

func (h *AuthHandler) GetAuthStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	email, ok := auth.GetUserEmailFromContext(ctx)
	if !ok {
		log.Println("AuthHandler: No user email in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	isSetupComplete, err := h.checkSetupComplete(ctx, email)
	if err != nil {
		log.Printf("AuthHandler: Failed to check setup status: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := models.AuthStatusResponse{
		IsAuthenticated: true, // TODO: Check if user is authenticated
		IsSetupComplete: isSetupComplete,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("AuthHandler: Failed to encode response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *AuthHandler) checkSetupComplete(ctx context.Context, email string) (bool, error) {
	userID, err := db.GetOrCreateUser(ctx, h.pool, email)
	if err != nil {
		return false, err
	}

	exists, err := db.UserSettingsExist(ctx, h.pool, userID)
	if err != nil {
		return false, err
	}

	return exists, nil
}
