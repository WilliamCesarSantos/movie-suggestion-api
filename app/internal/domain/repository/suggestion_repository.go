package repository

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion/app/internal/domain/entity"
)

type SuggestionRepository interface {
	FindPopular(ctx context.Context, userID string, limit int, minRating float64) ([]*entity.Movie, error)
	FindContentBased(ctx context.Context, userID string, limit int, minRating float64) ([]*entity.Movie, error)
	FindCollaborative(ctx context.Context, userID string, limit int, minRating float64) ([]*entity.Movie, error)
	FindHybrid(ctx context.Context, userID string, limit int, minRating float64, contentWeight, collaborativeWeight float64) ([]*entity.Movie, error)
	FindSerendipity(ctx context.Context, userID string, limit int, minRating float64) ([]*entity.Movie, error)
}
