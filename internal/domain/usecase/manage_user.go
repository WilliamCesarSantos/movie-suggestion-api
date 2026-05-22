package usecase

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion/internal/domain/entity"
)

type ManageUserUseCase interface {
	Create(ctx context.Context, user *entity.User) error
	GetByID(ctx context.Context, id string) (*entity.User, error)
	RecordWatched(ctx context.Context, userID, movieID string, userRating float64, reaction string) (*entity.User, error)
}
