package usecase

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion/app/internal/domain/entity"
)

type UpdateUserProfileUseCase interface {
	Execute(ctx context.Context, user *entity.User) error
}
