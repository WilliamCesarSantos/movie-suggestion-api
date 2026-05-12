package usecase

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion/internal/domain/entity"
)

type UpdateUserProfileUseCase interface {
	Execute(ctx context.Context, user *entity.User) error
}
