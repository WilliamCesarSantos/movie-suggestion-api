package usecase

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion/internal/domain/entity"
)

type ManageUserUseCase interface {
	Create(ctx context.Context, user *entity.User) error
	GetByID(ctx context.Context, id string) (*entity.User, error)
	RecordWatched(ctx context.Context, userID, movieID string, rating float64) (*entity.User, error)
	RecordLiked(ctx context.Context, userID, movieID string, suggestionAlgorithmUsed entity.SuggestionAlgorithm) (*entity.User, error)
	RecordDisliked(ctx context.Context, userID, movieID string) (*entity.User, error)
}
