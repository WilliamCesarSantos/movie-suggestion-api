package router

import (
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/http/handler"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/http/middleware"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/observability"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func NewRouter(
	userHandler *handler.UserHandler,
	movieHandler *handler.MovieHandler,
	importHandler *handler.ImportHandler,
	authHandler *handler.AuthHandler,
	healthHandler *handler.HealthHandler,
	authMiddleware *middleware.AuthMiddleware,
	metrics *observability.Metrics,
) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RequestID)
	r.Use(middleware.ObservabilityMiddleware(metrics))

	r.Get("/api/v1/health", healthHandler.Health)
	r.Post("/api/v1/login", authHandler.Login)

	r.Route("/api/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)

			r.With(middleware.RequireRole("users:write")).Post("/users", userHandler.CreateUser)
			r.With(middleware.RequireRole("users:write")).Patch("/users/{id}", userHandler.PatchUser)

			r.With(middleware.RequireRole("users:read"), middleware.RequireOwnerOrWildcard()).Get("/users/{id}", userHandler.GetUser)

			r.With(middleware.RequireRole("users:read")).Get("/users", userHandler.ListUsers)

			r.With(middleware.RequireRole("suggestions:read")).Get("/suggestions", userHandler.GetSuggestions)

			r.With(middleware.RequireRole("movies:read")).Get("/movies/{id}", movieHandler.GetMovie)
			r.With(middleware.RequireRole("movies-watch:write")).Post("/movies/{id}/watched", movieHandler.RecordWatched)

			r.With(middleware.RequireRole("movies:write")).Post("/movie-import", importHandler.TriggerImport)
		})
	})

	return r
}
