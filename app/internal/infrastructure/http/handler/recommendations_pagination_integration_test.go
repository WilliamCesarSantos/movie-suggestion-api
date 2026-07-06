package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/auth"
	cursorinfra "github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/cursor"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/http/handler"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/http/middleware"
	"github.com/go-chi/chi/v5"
)

type paginationRecommendMoviesUseCase struct {
	movies        []*entity.Movie
	receivedEmail string
	receivedLimit int
	receivedAlgo  *entity.RecommendationAlgorithm
	receivedTitle string
}

func (uc *paginationRecommendMoviesUseCase) Execute(ctx context.Context, userEmail string, limit int, algorithmOverride *entity.RecommendationAlgorithm, title string) ([]*entity.Movie, error) {
	uc.receivedEmail = userEmail
	uc.receivedLimit = limit
	uc.receivedAlgo = algorithmOverride
	uc.receivedTitle = title
	return uc.movies, nil
}

func buildMoviesRecommendationsRouter(jwtService *auth.JWTService, recommendUC *paginationRecommendMoviesUseCase) *chi.Mux {
	manageUC := &integrationManageUserUseCase{}
	h := handler.NewUserHandler(manageUC, recommendUC, nil, nil, nil, nil, "test-secret", 50)
	authMiddleware := middleware.NewAuthMiddleware(jwtService)

	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authMiddleware.Authenticate)
		r.With(middleware.RequireAnyRole("movies:read", "movies:write")).Get("/movies", h.GetRecommendedMovies)
	})
	return r
}

func moviesAuthHeader(t *testing.T, jwtService *auth.JWTService) string {
	t.Helper()
	token, _, err := jwtService.Generate("user-id", "alice@example.com", []string{"movies:read"})
	if err != nil {
		t.Fatalf("token generation failed: %v", err)
	}
	return "Bearer " + token
}

func TestGetMoviesRecommendations_FirstPageReturnsPaginationMetadata(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 1)
	recommendUC := &paginationRecommendMoviesUseCase{movies: []*entity.Movie{
		{ID: "movie-1", Title: "Movie 1", Year: "2024", Poster: "p1", ImdbRating: 7.1},
		{ID: "movie-2", Title: "Movie 2", Year: "2023", Poster: "p2", ImdbRating: 7.2},
		{ID: "movie-3", Title: "Movie 3", Year: "2022", Poster: "p3", ImdbRating: 7.3},
	}}
	r := buildMoviesRecommendationsRouter(jwtService, recommendUC)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/movies?limit=2", nil)
	req.Header.Set("Authorization", moviesAuthHeader(t, jwtService))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if recommendUC.receivedLimit != 50 {
		t.Fatalf("expected use case limit 50, got %d", recommendUC.receivedLimit)
	}

	var resp struct {
		Data       []map[string]any `json:"data"`
		NextCursor *string          `json:"nextCursor"`
		PrevCursor *string          `json:"prevCursor"`
		HasNext    bool             `json:"hasNext"`
		HasPrev    bool             `json:"hasPrev"`
		Limit      int              `json:"limit"`
		Count      int              `json:"count"`
		Total      int              `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Data) != 2 || resp.Count != 2 || resp.Total != 3 || resp.Limit != 2 {
		t.Fatalf("unexpected pagination payload: %+v", resp)
	}
	if !resp.HasNext || resp.HasPrev {
		t.Fatalf("expected hasNext=true and hasPrev=false, got hasNext=%v hasPrev=%v", resp.HasNext, resp.HasPrev)
	}
	if resp.NextCursor == nil || *resp.NextCursor == "" {
		t.Fatal("expected nextCursor")
	}
	if resp.PrevCursor != nil {
		t.Fatalf("expected nil prevCursor, got %v", *resp.PrevCursor)
	}
}

func TestGetMoviesRecommendations_NextPageReturnsPrevCursor(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 1)
	recommendUC := &paginationRecommendMoviesUseCase{movies: []*entity.Movie{
		{ID: "movie-1", Title: "Movie 1"},
		{ID: "movie-2", Title: "Movie 2"},
		{ID: "movie-3", Title: "Movie 3"},
	}}
	r := buildMoviesRecommendationsRouter(jwtService, recommendUC)

	cursor := cursorinfra.Encode("test-secret", cursorinfra.Cursor{Offset: 2, Total: 3})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/movies?limit=2&cursor="+cursor, nil)
	req.Header.Set("Authorization", moviesAuthHeader(t, jwtService))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data       []map[string]any `json:"data"`
		NextCursor *string          `json:"nextCursor"`
		PrevCursor *string          `json:"prevCursor"`
		HasNext    bool             `json:"hasNext"`
		HasPrev    bool             `json:"hasPrev"`
		Count      int              `json:"count"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Data) != 1 || resp.Count != 1 {
		t.Fatalf("expected 1 item on second page, got %+v", resp)
	}
	if resp.HasNext || !resp.HasPrev {
		t.Fatalf("expected hasNext=false and hasPrev=true, got hasNext=%v hasPrev=%v", resp.HasNext, resp.HasPrev)
	}
	if resp.NextCursor != nil {
		t.Fatal("expected nil nextCursor")
	}
	if resp.PrevCursor == nil || *resp.PrevCursor == "" {
		t.Fatal("expected prevCursor")
	}
}

func TestGetMoviesRecommendations_InvalidCursorReturnsBadRequest(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 1)
	recommendUC := &paginationRecommendMoviesUseCase{movies: []*entity.Movie{}}
	r := buildMoviesRecommendationsRouter(jwtService, recommendUC)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/movies?cursor=invalid", nil)
	req.Header.Set("Authorization", moviesAuthHeader(t, jwtService))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetMoviesRecommendations_InvalidLimitReturnsBadRequest(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 1)
	recommendUC := &paginationRecommendMoviesUseCase{movies: []*entity.Movie{}}
	r := buildMoviesRecommendationsRouter(jwtService, recommendUC)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/movies?limit=0", nil)
	req.Header.Set("Authorization", moviesAuthHeader(t, jwtService))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetMoviesRecommendations_TitleFilterIsPassedThrough(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 1)
	recommendUC := &paginationRecommendMoviesUseCase{movies: []*entity.Movie{{ID: "movie-1", Title: "Matrix"}}}
	r := buildMoviesRecommendationsRouter(jwtService, recommendUC)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/movies?title=mat", nil)
	req.Header.Set("Authorization", moviesAuthHeader(t, jwtService))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if recommendUC.receivedTitle != "mat" {
		t.Fatalf("expected title filter 'mat', got %q", recommendUC.receivedTitle)
	}
}

func TestGetMoviesRecommendations_WithoutMoviesReadOrWriteReturnsForbidden(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 1)
	recommendUC := &paginationRecommendMoviesUseCase{movies: []*entity.Movie{}}
	r := buildMoviesRecommendationsRouter(jwtService, recommendUC)

	token, _, err := jwtService.Generate("user-id", "alice@example.com", []string{"users:read"})
	if err != nil {
		t.Fatalf("token generation failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/movies", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}
