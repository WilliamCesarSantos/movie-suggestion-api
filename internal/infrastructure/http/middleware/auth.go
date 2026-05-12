package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/WilliamCesarSantos/movie-suggestion/internal/infrastructure/lambda"
	"github.com/go-chi/chi/v5"
)

type contextKey string

const (
	ContextKeyUserID contextKey = "userId"
	ContextKeyRole   contextKey = "role"
)

type AuthMiddleware struct {
	authClient *lambda.AuthClient
}

func NewAuthMiddleware(authClient *lambda.AuthClient) *AuthMiddleware {
	return &AuthMiddleware{authClient: authClient}
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")
		resp, err := m.authClient.Validate(r.Context(), token)
		if err != nil || !resp.Valid {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), ContextKeyUserID, resp.UserID)
		ctx = context.WithValue(ctx, ContextKeyRole, resp.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) AuthorizeUserOrAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, _ := r.Context().Value(ContextKeyRole).(string)
		if role == "admin" {
			next.ServeHTTP(w, r)
			return
		}
		tokenUserID, _ := r.Context().Value(ContextKeyUserID).(string)
		pathUserID := chi.URLParam(r, "id")
		if tokenUserID != pathUserID {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *AuthMiddleware) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, _ := r.Context().Value(ContextKeyRole).(string)
		if role != "admin" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
