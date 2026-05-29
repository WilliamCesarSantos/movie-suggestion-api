package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/WilliamCesarSantos/movie-suggestion/app/internal/infrastructure/auth"
)

type contextKey string

const (
	ContextKeyUserID contextKey = "userId"
	ContextKeyRoles  contextKey = "roles"
)

type AuthMiddleware struct {
	jwtService *auth.JWTService
}

func NewAuthMiddleware(jwtService *auth.JWTService) *AuthMiddleware {
	return &AuthMiddleware{jwtService: jwtService}
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := m.jwtService.Validate(token)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), ContextKeyUserID, claims.Subject)
		ctx = context.WithValue(ctx, ContextKeyRoles, claims.Roles)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
