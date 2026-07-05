package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
	domainusecase "github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/usecase"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/auth"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/http/handler"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/http/middleware"
	"github.com/go-chi/chi/v5"
)

type integrationLoginUseCase struct {
	jwtService *auth.JWTService
	email      string
	roles      []string
}

func (uc *integrationLoginUseCase) Execute(ctx context.Context, email, password string) (*domainusecase.LoginResult, error) {
	token, expiresAt, err := uc.jwtService.Generate("postgres-user-id", uc.email, uc.roles)
	if err != nil {
		return nil, err
	}
	return &domainusecase.LoginResult{
		Token:     token,
		Email:     uc.email,
		Roles:     uc.roles,
		ExpiresAt: expiresAt,
	}, nil
}

type integrationSuggestMoviesUseCase struct {
	receivedEmail string
}

func (uc *integrationSuggestMoviesUseCase) Execute(ctx context.Context, userEmail string, limit int, algorithmOverride *entity.SuggestionAlgorithm) ([]*entity.Movie, error) {
	uc.receivedEmail = userEmail
	return []*entity.Movie{{
		ID:         "movie-1",
		Title:      "Movie 1",
		Year:       "2024",
		Poster:     "https://example.com/poster.jpg",
		ImdbRating: 8.1,
	}}, nil
}

type integrationManageUserUseCase struct{}

func (uc *integrationManageUserUseCase) Create(ctx context.Context, user *entity.User) error {
	return nil
}

func (uc *integrationManageUserUseCase) GetByID(ctx context.Context, id string) (*entity.User, error) {
	return nil, entity.ErrUserNotFound
}

func (uc *integrationManageUserUseCase) RecordWatched(ctx context.Context, userID, movieID string, userRating float64, reaction string) (*entity.User, error) {
	return nil, nil
}

func TestIntegration_LoginThenSuggestions_UsesEmailFromToken(t *testing.T) {
	const expectedEmail = "william_cesar_santos@hotmail.com"

	jwtService := auth.NewJWTService("integration-secret", 1)
	loginUC := &integrationLoginUseCase{
		jwtService: jwtService,
		email:      expectedEmail,
		roles:      []string{"suggestions:read"},
	}
	suggestUC := &integrationSuggestMoviesUseCase{}
	manageUC := &integrationManageUserUseCase{}

	authHandler := handler.NewAuthHandler(loginUC)
	userHandler := handler.NewUserHandler(manageUC, suggestUC, nil, nil, nil, "integration-secret", 50)
	authMiddleware := middleware.NewAuthMiddleware(jwtService)

	r := chi.NewRouter()
	r.Post("/api/v1/login", authHandler.Login)
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authMiddleware.Authenticate)
		r.With(middleware.RequireRole("suggestions:read")).Get("/suggestions", userHandler.GetSuggestions)
	})

	loginBody := map[string]string{
		"email":    expectedEmail,
		"password": "123456",
	}
	loginJSON, err := json.Marshal(loginBody)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(loginJSON))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	r.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d", loginRec.Code)
	}

	var loginResp struct {
		Token     string   `json:"token"`
		Email     string   `json:"email"`
		Roles     []string `json:"roles"`
		ExpiresAt string   `json:"expiresAt"`
	}
	if err := json.Unmarshal(loginRec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("json.Unmarshal(login response) error = %v", err)
	}
	if loginResp.Token == "" {
		t.Fatal("expected token in login response")
	}

	claims, err := jwtService.Validate(loginResp.Token)
	if err != nil {
		t.Fatalf("Validate(token) error = %v", err)
	}
	if claims.Email != expectedEmail {
		t.Fatalf("expected token email %s, got %s", expectedEmail, claims.Email)
	}
	if claims.ExpiresAt == nil || claims.ExpiresAt.Time.Before(time.Now()) {
		t.Fatalf("expected token with future expiration, got %#v", claims.ExpiresAt)
	}

	suggestionsReq := httptest.NewRequest(http.MethodGet, "/api/v1/suggestions", nil)
	suggestionsReq.Header.Set("Authorization", "Bearer "+loginResp.Token)
	suggestionsRec := httptest.NewRecorder()
	r.ServeHTTP(suggestionsRec, suggestionsReq)

	if suggestionsRec.Code != http.StatusOK {
		t.Fatalf("expected suggestions status 200, got %d body=%s", suggestionsRec.Code, suggestionsRec.Body.String())
	}
	if suggestUC.receivedEmail != expectedEmail {
		t.Fatalf("expected suggest use case to receive email %s, got %s", expectedEmail, suggestUC.receivedEmail)
	}

	var suggestionsResp struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(suggestionsRec.Body.Bytes(), &suggestionsResp); err != nil {
		t.Fatalf("json.Unmarshal(suggestions response) error = %v", err)
	}
	if len(suggestionsResp.Data) != 1 {
		t.Fatalf("expected 1 suggested movie, got %d", len(suggestionsResp.Data))
	}
}
