package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/repository"
	domainusecase "github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/usecase"
	appusecase "github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/application/usecase"
)

type mockAuthUserRepository struct {
	listFn func(ctx context.Context, filters repository.AuthUserFilters) ([]*entity.AuthUser, int, error)
}

func (m *mockAuthUserRepository) Create(ctx context.Context, user *entity.AuthUser) error {
	return nil
}

func (m *mockAuthUserRepository) FindByEmail(ctx context.Context, email string) (*entity.AuthUser, error) {
	return nil, entity.ErrAuthUserNotFound
}

func (m *mockAuthUserRepository) List(ctx context.Context, filters repository.AuthUserFilters) ([]*entity.AuthUser, int, error) {
	return m.listFn(ctx, filters)
}

func makeUser(id, name, email string) *entity.AuthUser {
	return &entity.AuthUser{ID: id, Name: name, Email: email, Roles: []string{"users:read"}, CreatedAt: time.Now()}
}

func TestListUsersUseCase_ReadOnly_ForcesOwnEmail(t *testing.T) {
	callerEmail := "alice@example.com"
	alice := makeUser("id-alice", "Alice", callerEmail)
	bob := makeUser("id-bob", "Bob", "bob@example.com")
	all := []*entity.AuthUser{alice, bob}

	repo := &mockAuthUserRepository{
		listFn: func(ctx context.Context, filters repository.AuthUserFilters) ([]*entity.AuthUser, int, error) {
			if filters.Email != callerEmail {
				t.Errorf("expected forced email filter %q, got %q", callerEmail, filters.Email)
			}
			var result []*entity.AuthUser
			for _, u := range all {
				if u.Email == filters.Email {
					result = append(result, u)
				}
			}
			return result, len(result), nil
		},
	}

	uc := appusecase.NewListUsersUseCase(repo)
	out, err := uc.Execute(context.Background(), callerEmail, false, domainusecase.ListUsersInput{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(out.Users))
	}
	if out.Users[0].Email != callerEmail {
		t.Errorf("expected email %q, got %q", callerEmail, out.Users[0].Email)
	}
}

func TestListUsersUseCase_WriteRole_AllowsMultiple(t *testing.T) {
	callerEmail := "admin@example.com"
	users := []*entity.AuthUser{
		makeUser("id-1", "Alice", "alice@example.com"),
		makeUser("id-2", "Bob", "bob@example.com"),
	}

	repo := &mockAuthUserRepository{
		listFn: func(ctx context.Context, filters repository.AuthUserFilters) ([]*entity.AuthUser, int, error) {
			if filters.Email != "" {
				t.Errorf("expected no forced email filter for write role, got %q", filters.Email)
			}
			return users, len(users), nil
		},
	}

	uc := appusecase.NewListUsersUseCase(repo)
	out, err := uc.Execute(context.Background(), callerEmail, true, domainusecase.ListUsersInput{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(out.Users))
	}
}

func TestListUsersUseCase_WriteRole_WithEmailFilter(t *testing.T) {
	callerEmail := "admin@example.com"
	targetEmail := "alice@example.com"
	alice := makeUser("id-1", "Alice", targetEmail)

	repo := &mockAuthUserRepository{
		listFn: func(ctx context.Context, filters repository.AuthUserFilters) ([]*entity.AuthUser, int, error) {
			if filters.Email != targetEmail {
				t.Errorf("expected email filter %q, got %q", targetEmail, filters.Email)
			}
			return []*entity.AuthUser{alice}, 1, nil
		},
	}

	uc := appusecase.NewListUsersUseCase(repo)
	out, err := uc.Execute(context.Background(), callerEmail, true, domainusecase.ListUsersInput{Email: targetEmail, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(out.Users))
	}
}

func TestListUsersUseCase_DefaultPagination(t *testing.T) {
	repo := &mockAuthUserRepository{
		listFn: func(ctx context.Context, filters repository.AuthUserFilters) ([]*entity.AuthUser, int, error) {
			return nil, 0, nil
		},
	}
	uc := appusecase.NewListUsersUseCase(repo)
	out, err := uc.Execute(context.Background(), "x@x.com", true, domainusecase.ListUsersInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Page != 1 {
		t.Errorf("expected page 1, got %d", out.Page)
	}
	if out.PageSize != 20 {
		t.Errorf("expected pageSize 20, got %d", out.PageSize)
	}
}
