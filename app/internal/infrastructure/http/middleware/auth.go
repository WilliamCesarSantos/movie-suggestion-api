package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/auth"
	"github.com/rs/zerolog/log"
)

type contextKey string

const (
	ContextKeyUserID   contextKey = "userId"
	ContextKeyRoles    contextKey = "roles"
	ContextKeyUsername contextKey = "username"
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
			log.Ctx(r.Context()).Warn().Msg("missing or malformed authorization header")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := m.jwtService.Validate(token)
		if err != nil {
			log.Ctx(r.Context()).Warn().Err(err).Msg("invalid JWT token")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), ContextKeyUserID, claims.Subject)
		ctx = context.WithValue(ctx, ContextKeyRoles, claims.Roles)
		ctx = context.WithValue(ctx, ContextKeyUsername, claims.Email)
		logger := log.Ctx(ctx).With().Str("username", claims.Email).Logger()
		ctx = logger.WithContext(ctx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
