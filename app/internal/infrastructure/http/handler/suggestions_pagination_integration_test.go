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

type paginationSuggestMoviesUseCase struct {
	movies        []*entity.Movie
	receivedEmail string
	receivedLimit int
	receivedAlgo  *entity.SuggestionAlgorithm
}

func (uc *paginationSuggestMoviesUseCase) Execute(ctx context.Context, userEmail string, limit int, algorithmOverride *entity.SuggestionAlgorithm) ([]*entity.Movie, error) {
	uc.receivedEmail = userEmail
	uc.receivedLimit = limit
	uc.receivedAlgo = algorithmOverride
	return uc.movies, nil
}

func buildSuggestionsRouter(jwtService *auth.JWTService, suggestUC *paginationSuggestMoviesUseCase) *chi.Mux {
	manageUC := &integrationManageUserUseCase{}
	h := handler.NewUserHandler(manageUC, suggestUC, nil, nil, nil, nil, "test-secret", 50)
	authMiddleware := middleware.NewAuthMiddleware(jwtService)

	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authMiddleware.Authenticate)
		r.With(middleware.RequireRole("suggestions:read")).Get("/suggestions", h.GetSuggestions)
	})
	return r
}

func suggestionsAuthHeader(t *testing.T, jwtService *auth.JWTService) string {
	t.Helper()
	token, _, err := jwtService.Generate("user-id", "alice@example.com", []string{"suggestions:read"})
	if err != nil {
		t.Fatalf("token generation failed: %v", err)
	}
	return "Bearer " + token
}

func TestGetSuggestions_FirstPageReturnsPaginationMetadata(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 1)
	suggestUC := &paginationSuggestMoviesUseCase{movies: []*entity.Movie{
		{ID: "movie-1", Title: "Movie 1", Year: "2024", Poster: "p1", ImdbRating: 7.1},
		{ID: "movie-2", Title: "Movie 2", Year: "2023", Poster: "p2", ImdbRating: 7.2},
		{ID: "movie-3", Title: "Movie 3", Year: "2022", Poster: "p3", ImdbRating: 7.3},
	}}
	r := buildSuggestionsRouter(jwtService, suggestUC)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/suggestions?limit=2", nil)
	req.Header.Set("Authorization", suggestionsAuthHeader(t, jwtService))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if suggestUC.receivedLimit != 50 {
		t.Fatalf("expected use case limit 50, got %d", suggestUC.receivedLimit)
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

func TestGetSuggestions_NextPageReturnsPrevCursor(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 1)
	suggestUC := &paginationSuggestMoviesUseCase{movies: []*entity.Movie{
		{ID: "movie-1", Title: "Movie 1"},
		{ID: "movie-2", Title: "Movie 2"},
		{ID: "movie-3", Title: "Movie 3"},
	}}
	r := buildSuggestionsRouter(jwtService, suggestUC)

	cursor := cursorinfra.Encode("test-secret", cursorinfra.Cursor{Offset: 2, Total: 3})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/suggestions?limit=2&cursor="+cursor, nil)
	req.Header.Set("Authorization", suggestionsAuthHeader(t, jwtService))
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

func TestGetSuggestions_InvalidCursorReturnsBadRequest(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 1)
	suggestUC := &paginationSuggestMoviesUseCase{movies: []*entity.Movie{}}
	r := buildSuggestionsRouter(jwtService, suggestUC)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/suggestions?cursor=invalid", nil)
	req.Header.Set("Authorization", suggestionsAuthHeader(t, jwtService))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetSuggestions_InvalidLimitReturnsBadRequest(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 1)
	suggestUC := &paginationSuggestMoviesUseCase{movies: []*entity.Movie{}}
	r := buildSuggestionsRouter(jwtService, suggestUC)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/suggestions?limit=0", nil)
	req.Header.Set("Authorization", suggestionsAuthHeader(t, jwtService))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}
