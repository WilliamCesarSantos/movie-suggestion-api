package usecase

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
)

type UpdateUserProfileUseCase interface {
	Execute(ctx context.Context, user *entity.User) error
}
