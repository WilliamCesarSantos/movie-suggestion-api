package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/WilliamCesarSantos/movie-suggestion/app/internal/domain/entity"
	domainusecase "github.com/WilliamCesarSantos/movie-suggestion/app/internal/domain/usecase"
)

type AuthHandler struct {
	loginUC domainusecase.LoginUseCase
}

func NewAuthHandler(loginUC domainusecase.LoginUseCase) *AuthHandler {
	return &AuthHandler{loginUC: loginUC}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token     string   `json:"token"`
	Email     string   `json:"email"`
	Roles     []string `json:"roles"`
	ExpiresAt string   `json:"expiresAt"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" {
		http.Error(w, "email and password required", http.StatusBadRequest)
		return
	}
	result, err := h.loginUC.Execute(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, entity.ErrAuthUserNotFound) {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(loginResponse{
		Token:     result.Token,
		Email:     result.Email,
		Roles:     result.Roles,
		ExpiresAt: result.ExpiresAt.Format(time.RFC3339),
	})
}
