package repository

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
)

type AuthUserFilters struct {
	Email    string
	Name     string
	Page     int
	PageSize int
}

type AuthUserUpdate struct {
	Name     *string
	Password *string
	Roles    *[]string
}

type AuthUserRepository interface {
	Create(ctx context.Context, user *entity.AuthUser) error
	FindByID(ctx context.Context, id string) (*entity.AuthUser, error)
	FindByEmail(ctx context.Context, email string) (*entity.AuthUser, error)
	List(ctx context.Context, filters AuthUserFilters) ([]*entity.AuthUser, int, error)
	Update(ctx context.Context, id string, update AuthUserUpdate) error
}
