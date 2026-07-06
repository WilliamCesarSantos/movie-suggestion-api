package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

type stubPatchUserUseCase struct {
	captured *domainusecase.PatchUserInput
	out      *domainusecase.PatchUserOutput
	err      error
}

func (s *stubPatchUserUseCase) Execute(ctx context.Context, input domainusecase.PatchUserInput) (*domainusecase.PatchUserOutput, error) {
	s.captured = &input
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &domainusecase.PatchUserOutput{
		ID:        input.TargetUserID,
		Name:      "Updated",
		Email:     "user@example.com",
		Roles:     []string{"users:read"},
		CreatedAt: time.Now().Format(time.RFC3339),
	}, nil
}

func buildPatchUserRouter(jwtService *auth.JWTService, patchUC domainusecase.PatchUserUseCase) *chi.Mux {
	manageUC := &integrationManageUserUseCase{}
	suggestUC := &integrationSuggestMoviesUseCase{}
	h := handler.NewUserHandler(manageUC, suggestUC, nil, patchUC, nil, nil, "test-secret", 50)
	authMiddleware := middleware.NewAuthMiddleware(jwtService)

	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authMiddleware.Authenticate)
		r.With(middleware.RequireRole("users:write")).Patch("/users/{id}", h.PatchUser)
	})
	return r
}

func TestPatchUser_OwnerSuccess(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 1)
	token, _, err := jwtService.Generate("owner-id", "owner@example.com", []string{"users:write"})
	if err != nil {
		t.Fatalf("token generation failed: %v", err)
	}

	patchUC := &stubPatchUserUseCase{}
	r := buildPatchUserRouter(jwtService, patchUC)

	body := map[string]any{
		"name":     "New Name",
		"password": "123456",
		"roles":    []string{"users:read", "movies:read"},
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/owner-id", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if patchUC.captured == nil {
		t.Fatal("expected use case invocation")
	}
	if patchUC.captured.TargetUserID != "owner-id" || patchUC.captured.CallerUserID != "owner-id" {
		t.Fatalf("unexpected IDs: target=%s caller=%s", patchUC.captured.TargetUserID, patchUC.captured.CallerUserID)
	}
}

func TestPatchUser_RequiresUsersWriteRole(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 1)
	token, _, err := jwtService.Generate("owner-id", "owner@example.com", []string{"users:read"})
	if err != nil {
		t.Fatalf("token generation failed: %v", err)
	}

	patchUC := &stubPatchUserUseCase{}
	r := buildPatchUserRouter(jwtService, patchUC)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/owner-id", bytes.NewReader([]byte(`{"roles":[]}`)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestPatchUser_MapsDomainErrors(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 1)
	token, _, err := jwtService.Generate("caller-id", "caller@example.com", []string{"users:write"})
	if err != nil {
		t.Fatalf("token generation failed: %v", err)
	}

	tests := []struct {
		name       string
		usecaseErr error
		expected   int
	}{
		{name: "invalid", usecaseErr: entity.ErrInvalidUserPatchInput, expected: http.StatusBadRequest},
		{name: "forbidden", usecaseErr: entity.ErrUserPatchForbidden, expected: http.StatusForbidden},
		{name: "not found", usecaseErr: entity.ErrAuthUserNotFound, expected: http.StatusNotFound},
		{name: "internal", usecaseErr: errors.New("boom"), expected: http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			patchUC := &stubPatchUserUseCase{err: tc.usecaseErr}
			r := buildPatchUserRouter(jwtService, patchUC)

			req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/target-id", bytes.NewReader([]byte(`{"roles":[]}`)))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tc.expected {
				t.Fatalf("expected %d, got %d body=%s", tc.expected, rec.Code, rec.Body.String())
			}
		})
	}
}
