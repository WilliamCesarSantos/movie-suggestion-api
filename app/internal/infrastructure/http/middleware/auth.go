package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/auth"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type contextKey string

const (
	ContextKeyUserID        contextKey = "userId"
	ContextKeyUserEmail     contextKey = "userEmail"
	ContextKeyRoles         contextKey = "roles"
	ContextKeyCorrelationID contextKey = "correlationId"
)

type AuthMiddleware struct {
	jwtService *auth.JWTService
}

func NewAuthMiddleware(jwtService *auth.JWTService) *AuthMiddleware {
	return &AuthMiddleware{jwtService: jwtService}
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := log.Ctx(r.Context()).With().Str("logger", "http.auth").Logger()
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			logger.Warn().Msg("missing or malformed authorization header")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := m.jwtService.Validate(token)
		if err != nil {
			logger.Warn().Err(err).Msg("invalid JWT token")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if claims.Email == "" {
			logger.Warn().Msg("JWT token missing email claim")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		// Update the request logger in place (not a copy) so every log of this request,
		// including the access log written by ObservabilityMiddleware, carries the user.
		log.Ctx(r.Context()).UpdateContext(func(c zerolog.Context) zerolog.Context {
			return c.Str("userEmail", claims.Email)
		})
		ctx := context.WithValue(r.Context(), ContextKeyUserID, claims.Subject)
		ctx = context.WithValue(ctx, ContextKeyUserEmail, claims.Email)
		ctx = context.WithValue(ctx, ContextKeyRoles, claims.Roles)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
