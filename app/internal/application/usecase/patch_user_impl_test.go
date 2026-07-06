package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/repository"
	domainusecase "github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/usecase"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/auth"
)

type mockPatchAuthUserRepo struct {
	user      *entity.AuthUser
	findErr   error
	updateErr error
	update    repository.AuthUserUpdate
}

func (m *mockPatchAuthUserRepo) Create(ctx context.Context, user *entity.AuthUser) error { return nil }
func (m *mockPatchAuthUserRepo) FindByID(ctx context.Context, id string) (*entity.AuthUser, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	if m.user == nil {
		return nil, entity.ErrAuthUserNotFound
	}
	copyUser := *m.user
	copyUser.Roles = append([]string(nil), m.user.Roles...)
	return &copyUser, nil
}
func (m *mockPatchAuthUserRepo) FindByEmail(ctx context.Context, email string) (*entity.AuthUser, error) {
	if m.user == nil {
		return nil, entity.ErrAuthUserNotFound
	}
	copyUser := *m.user
	copyUser.Roles = append([]string(nil), m.user.Roles...)
	return &copyUser, nil
}
func (m *mockPatchAuthUserRepo) List(ctx context.Context, filters repository.AuthUserFilters) ([]*entity.AuthUser, int, error) {
	return nil, 0, nil
}
func (m *mockPatchAuthUserRepo) Update(ctx context.Context, id string, update repository.AuthUserUpdate) error {
	m.update = update
	if m.updateErr != nil {
		return m.updateErr
	}
	if m.user == nil {
		return entity.ErrAuthUserNotFound
	}
	if update.Name != nil {
		m.user.Name = *update.Name
	}
	if update.Password != nil {
		m.user.Password = *update.Password
	}
	if update.Roles != nil {
		m.user.Roles = append([]string(nil), (*update.Roles)...)
	}
	return nil
}

type mockPatchGraphUserRepo struct {
	userByID      *entity.User
	userByEmail   *entity.User
	findByIDErr   error
	findByMailErr error
	updated       *entity.User
}

func (m *mockPatchGraphUserRepo) Create(ctx context.Context, user *entity.User) error { return nil }
func (m *mockPatchGraphUserRepo) FindByID(ctx context.Context, id string) (*entity.User, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	if m.userByID == nil {
		return nil, entity.ErrUserNotFound
	}
	copyUser := *m.userByID
	return &copyUser, nil
}
func (m *mockPatchGraphUserRepo) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	if m.findByMailErr != nil {
		return nil, m.findByMailErr
	}
	if m.userByEmail == nil {
		return nil, entity.ErrUserNotFound
	}
	copyUser := *m.userByEmail
	return &copyUser, nil
}
func (m *mockPatchGraphUserRepo) UpdateProfile(ctx context.Context, user *entity.User) error {
	copyUser := *user
	m.updated = &copyUser
	return nil
}
func (m *mockPatchGraphUserRepo) RecordWatched(ctx context.Context, userID, movieID string, userRating float64, reaction string) error {
	return nil
}

func TestPatchUserUseCase_OwnerCanUpdateAllFields(t *testing.T) {
	passwordService := auth.NewPasswordService("pepper")
	authRepo := &mockPatchAuthUserRepo{user: &entity.AuthUser{
		ID:        "user-1",
		Name:      "Old Name",
		Email:     "user@example.com",
		Password:  "oldhash",
		Roles:     []string{"users:read"},
		CreatedAt: time.Now(),
	}}
	graphRepo := &mockPatchGraphUserRepo{userByID: &entity.User{ID: "user-1", Name: "Old Name", Email: "user@example.com", CurrentAlgorithm: entity.AlgorithmPopular}}
	uc := NewPatchUserUseCase(authRepo, graphRepo, passwordService)

	newName := "New Name"
	newPassword := "123456"
	newRoles := []string{"users:write", "movies:read"}
	out, err := uc.Execute(context.Background(), domainusecase.PatchUserInput{
		TargetUserID: "user-1",
		CallerUserID: "user-1",
		Name:         &newName,
		Password:     &newPassword,
		Roles:        &newRoles,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if out.Name != "New Name" {
		t.Fatalf("expected updated name, got %s", out.Name)
	}
	if len(out.Roles) != 2 {
		t.Fatalf("expected 2 roles, got %d", len(out.Roles))
	}
	if authRepo.update.Password == nil || *authRepo.update.Password == "123456" {
		t.Fatalf("expected hashed password in update payload")
	}
	if graphRepo.updated == nil || graphRepo.updated.Name != "New Name" {
		t.Fatalf("expected graph profile update with new name")
	}
}

func TestPatchUserUseCase_NonOwnerCannotUpdateNameOrPassword(t *testing.T) {
	passwordService := auth.NewPasswordService("pepper")
	authRepo := &mockPatchAuthUserRepo{user: &entity.AuthUser{ID: "target", Email: "target@example.com", Roles: []string{"users:read"}, CreatedAt: time.Now()}}
	graphRepo := &mockPatchGraphUserRepo{}
	uc := NewPatchUserUseCase(authRepo, graphRepo, passwordService)

	name := "Blocked"
	_, err := uc.Execute(context.Background(), domainusecase.PatchUserInput{
		TargetUserID: "target",
		CallerUserID: "other",
		Name:         &name,
	})
	if !errors.Is(err, entity.ErrUserPatchForbidden) {
		t.Fatalf("expected ErrUserPatchForbidden, got %v", err)
	}
}

func TestPatchUserUseCase_NonOwnerCanUpdateRolesOnly(t *testing.T) {
	passwordService := auth.NewPasswordService("pepper")
	authRepo := &mockPatchAuthUserRepo{user: &entity.AuthUser{ID: "target", Email: "target@example.com", Roles: []string{"users:read"}, CreatedAt: time.Now()}}
	graphRepo := &mockPatchGraphUserRepo{}
	uc := NewPatchUserUseCase(authRepo, graphRepo, passwordService)

	roles := []string{"movies:write"}
	out, err := uc.Execute(context.Background(), domainusecase.PatchUserInput{
		TargetUserID: "target",
		CallerUserID: "other",
		Roles:        &roles,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(out.Roles) != 1 || out.Roles[0] != "movies:write" {
		t.Fatalf("unexpected roles: %#v", out.Roles)
	}
}

func TestPatchUserUseCase_Validations(t *testing.T) {
	passwordService := auth.NewPasswordService("pepper")
	authRepo := &mockPatchAuthUserRepo{user: &entity.AuthUser{ID: "target", Email: "target@example.com", Roles: []string{"users:read"}, CreatedAt: time.Now()}}
	graphRepo := &mockPatchGraphUserRepo{}
	uc := NewPatchUserUseCase(authRepo, graphRepo, passwordService)

	t.Run("empty body", func(t *testing.T) {
		_, err := uc.Execute(context.Background(), domainusecase.PatchUserInput{TargetUserID: "target", CallerUserID: "target"})
		if !errors.Is(err, entity.ErrInvalidUserPatchInput) {
			t.Fatalf("expected invalid patch input, got %v", err)
		}
	})

	t.Run("password too short", func(t *testing.T) {
		pwd := "12345"
		_, err := uc.Execute(context.Background(), domainusecase.PatchUserInput{TargetUserID: "target", CallerUserID: "target", Password: &pwd})
		if !errors.Is(err, entity.ErrInvalidUserPatchInput) {
			t.Fatalf("expected invalid patch input, got %v", err)
		}
	})

	t.Run("invalid role", func(t *testing.T) {
		roles := []string{"admin"}
		_, err := uc.Execute(context.Background(), domainusecase.PatchUserInput{TargetUserID: "target", CallerUserID: "target", Roles: &roles})
		if !errors.Is(err, entity.ErrInvalidUserPatchInput) {
			t.Fatalf("expected invalid patch input, got %v", err)
		}
	})
}

func TestPatchUserUseCase_ReturnsNotFound(t *testing.T) {
	passwordService := auth.NewPasswordService("pepper")
	authRepo := &mockPatchAuthUserRepo{findErr: entity.ErrAuthUserNotFound}
	graphRepo := &mockPatchGraphUserRepo{}
	uc := NewPatchUserUseCase(authRepo, graphRepo, passwordService)

	roles := []string{}
	_, err := uc.Execute(context.Background(), domainusecase.PatchUserInput{TargetUserID: "missing", CallerUserID: "missing", Roles: &roles})
	if !errors.Is(err, entity.ErrAuthUserNotFound) {
		t.Fatalf("expected ErrAuthUserNotFound, got %v", err)
	}
}
