package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/WilliamCesarSantos/movie-suggestion/internal/domain/entity"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/infrastructure/auth"
)

type mockAuthUserRepository struct {
	user *entity.AuthUser
	err  error
}

func (m *mockAuthUserRepository) Create(ctx context.Context, user *entity.AuthUser) error { return nil }
func (m *mockAuthUserRepository) FindByEmail(ctx context.Context, email string) (*entity.AuthUser, error) {
	return m.user, m.err
}

func TestLoginUseCase_Execute(t *testing.T) {
	passwordService := auth.NewPasswordService("pepper")
	jwtService := auth.NewJWTService("secret", 1)

	hashedPassword, err := passwordService.Hash("password123")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	repo := &mockAuthUserRepository{user: &entity.AuthUser{
		ID:       "user-1",
		Email:    "user@example.com",
		Password: hashedPassword,
		Roles:    []string{"users:read"},
	}}

	uc := NewLoginUseCase(repo, passwordService, jwtService)

	result, err := uc.Execute(context.Background(), "user@example.com", "password123")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Token == "" {
		t.Fatal("expected token")
	}
	if result.Email != "user@example.com" {
		t.Fatalf("expected email user@example.com, got %s", result.Email)
	}
	if len(result.Roles) != 1 || result.Roles[0] != "users:read" {
		t.Fatalf("unexpected roles: %#v", result.Roles)
	}
}

func TestLoginUseCase_ExecuteInvalidCredentials(t *testing.T) {
	passwordService := auth.NewPasswordService("pepper")
	jwtService := auth.NewJWTService("secret", 1)

	hashedPassword, err := passwordService.Hash("password123")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	repo := &mockAuthUserRepository{user: &entity.AuthUser{
		ID:       "user-1",
		Email:    "user@example.com",
		Password: hashedPassword,
		Roles:    []string{"users:read"},
	}}

	uc := NewLoginUseCase(repo, passwordService, jwtService)

	_, err = uc.Execute(context.Background(), "user@example.com", "wrong")
	if !errors.Is(err, entity.ErrAuthUserNotFound) {
		t.Fatalf("expected ErrAuthUserNotFound, got %v", err)
	}
}

func TestLoginUseCase_ExecuteUserNotFound(t *testing.T) {
	passwordService := auth.NewPasswordService("pepper")
	jwtService := auth.NewJWTService("secret", 1)
	repo := &mockAuthUserRepository{err: entity.ErrAuthUserNotFound}

	uc := NewLoginUseCase(repo, passwordService, jwtService)

	_, err := uc.Execute(context.Background(), "missing@example.com", "password123")
	if !errors.Is(err, entity.ErrAuthUserNotFound) {
		t.Fatalf("expected ErrAuthUserNotFound, got %v", err)
	}
}
