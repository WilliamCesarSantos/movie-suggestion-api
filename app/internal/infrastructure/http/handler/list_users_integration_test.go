package handler_test

import (
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

type stubListUsersUC struct {
	out *domainusecase.ListUsersOutput
	err error
}

func (s *stubListUsersUC) Execute(ctx context.Context, callerEmail string, callerHasWrite bool, input domainusecase.ListUsersInput) (*domainusecase.ListUsersOutput, error) {
	return s.out, s.err
}

func buildListUsersRouter(jwtService *auth.JWTService, listUC domainusecase.ListUsersUseCase) *chi.Mux {
	manageUC := &integrationManageUserUseCase{}
	recommendUC := &integrationRecommendMoviesUseCase{}
	h := handler.NewUserHandler(manageUC, recommendUC, listUC, nil, nil, nil, "test-secret", 50)
	authMiddleware := middleware.NewAuthMiddleware(jwtService)

	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authMiddleware.Authenticate)
		r.With(middleware.RequireRole("users:read")).Get("/users", h.ListUsers)
	})
	return r
}

func TestListUsers_ReadOnly_ReturnsSelf(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 1)
	email := "alice@example.com"
	token, _, err := jwtService.Generate("user-id-1", email, []string{"users:read"})
	if err != nil {
		t.Fatalf("token generation failed: %v", err)
	}

	now := time.Now()
	listUC := &stubListUsersUC{
		out: &domainusecase.ListUsersOutput{
			Users: []*entity.AuthUser{{
				ID: "user-id-1", Name: "Alice", Email: email,
				Roles: []string{"users:read"}, CreatedAt: now,
			}},
			Total:    1,
			Page:     1,
			PageSize: 20,
		},
	}

	r := buildListUsersRouter(jwtService, listUC)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data  []map[string]any `json:"data"`
		Total int              `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 user, got %d", len(resp.Data))
	}
	if resp.Total != 1 {
		t.Fatalf("expected total 1, got %d", resp.Total)
	}
}

func TestListUsers_NoRole_Returns403(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 1)
	token, _, err := jwtService.Generate("user-id-2", "bob@example.com", []string{"movies:read"})
	if err != nil {
		t.Fatalf("token generation failed: %v", err)
	}

	listUC := &stubListUsersUC{out: &domainusecase.ListUsersOutput{}}
	r := buildListUsersRouter(jwtService, listUC)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestListUsers_WriteRole_AllowsMultiple(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 1)
	email := "admin@example.com"
	token, _, err := jwtService.Generate("admin-id", email, []string{"users:read", "users:write"})
	if err != nil {
		t.Fatalf("token generation failed: %v", err)
	}

	now := time.Now()
	listUC := &stubListUsersUC{
		out: &domainusecase.ListUsersOutput{
			Users: []*entity.AuthUser{
				{ID: "id-1", Name: "Alice", Email: "alice@example.com", Roles: []string{"users:read"}, CreatedAt: now},
				{ID: "id-2", Name: "Bob", Email: "bob@example.com", Roles: []string{"users:read"}, CreatedAt: now},
			},
			Total:    2,
			Page:     1,
			PageSize: 20,
		},
	}

	r := buildListUsersRouter(jwtService, listUC)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data  []map[string]any `json:"data"`
		Total int              `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 users, got %d", len(resp.Data))
	}
}

func TestListUsers_InvalidPage_Returns400(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 1)
	token, _, err := jwtService.Generate("user-id", "u@e.com", []string{"users:read", "users:write"})
	if err != nil {
		t.Fatalf("token generation failed: %v", err)
	}

	listUC := &stubListUsersUC{out: &domainusecase.ListUsersOutput{}}
	r := buildListUsersRouter(jwtService, listUC)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?page=0", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
