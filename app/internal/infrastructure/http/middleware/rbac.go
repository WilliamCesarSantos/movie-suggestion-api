package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
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
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
