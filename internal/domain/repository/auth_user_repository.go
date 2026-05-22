package repository

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion/internal/domain/entity"
)

type AuthUserRepository interface {
	Create(ctx context.Context, user *entity.AuthUser) error
	FindByEmail(ctx context.Context, email string) (*entity.AuthUser, error)
}
