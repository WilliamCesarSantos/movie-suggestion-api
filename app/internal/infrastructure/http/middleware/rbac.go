package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

func hasRole(roles []string, required string) bool {
	for _, r := range roles {
		if r == "*" || r == required {
			return true
		}
	}
	return false
}

func hasWildcard(roles []string) bool {
	for _, r := range roles {
		if r == "*" {
			return true
		}
	}
	return false
}

func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roles, _ := r.Context().Value(ContextKeyRoles).([]string)
			if !hasRole(roles, role) {
				log.Ctx(r.Context()).Warn().Str("requiredRole", role).Msg("forbidden: missing role")
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireOwnerOrWildcard() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roles, _ := r.Context().Value(ContextKeyRoles).([]string)
			if hasWildcard(roles) {
				next.ServeHTTP(w, r)
				return
			}
			tokenUserID, _ := r.Context().Value(ContextKeyUserID).(string)
			pathID := chi.URLParam(r, "id")
			if tokenUserID != pathID {
				log.Ctx(r.Context()).Warn().Str("tokenUserId", tokenUserID).Str("pathUserId", pathID).Msg("forbidden: owner mismatch")
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
