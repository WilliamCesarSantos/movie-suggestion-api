package repository

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion/internal/domain/entity"
)

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	FindByID(ctx context.Context, id string) (*entity.User, error)
	UpdateProfile(ctx context.Context, user *entity.User) error
	RecordWatched(ctx context.Context, userID, movieID string, userRating float64, reaction string) error
}
