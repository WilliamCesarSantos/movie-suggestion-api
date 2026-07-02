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

type AuthUserRepository interface {
	Create(ctx context.Context, user *entity.AuthUser) error
	FindByEmail(ctx context.Context, email string) (*entity.AuthUser, error)
	List(ctx context.Context, filters AuthUserFilters) ([]*entity.AuthUser, int, error)
}
