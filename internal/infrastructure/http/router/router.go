package router

import (
	"github.com/WilliamCesarSantos/movie-suggestion/internal/infrastructure/http/handler"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/infrastructure/http/middleware"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/infrastructure/observability"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func NewRouter(
	userHandler *handler.UserHandler,
	movieHandler *handler.MovieHandler,
	adminHandler *handler.AdminHandler,
	healthHandler *handler.HealthHandler,
	authMiddleware *middleware.AuthMiddleware,
	metrics *observability.Metrics,
) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RequestID)
	r.Use(middleware.ObservabilityMiddleware(metrics))

	r.Get("/api/v1/health", healthHandler.Health)

	r.Route("/api/v1", func(r chi.Router) {
		// Public: create user
		r.Post("/users", userHandler.CreateUser)

		// Authenticated routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)

			r.Route("/users/{id}", func(r chi.Router) {
				r.Use(authMiddleware.AuthorizeUserOrAdmin)
				r.Get("/", userHandler.GetUser)
				r.Post("/watched", userHandler.RecordWatched)
				r.Post("/liked", userHandler.RecordLiked)
				r.Post("/disliked", userHandler.RecordDisliked)
				r.Get("/suggestions", userHandler.GetSuggestions)
			})

			r.Get("/movies/{id}", movieHandler.GetMovie)

			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireAdmin)
				r.Post("/admin/import/trigger", adminHandler.TriggerImport)
			})
		})
	})

	return r
}
