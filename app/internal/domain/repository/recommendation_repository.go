package repository

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
)

type RecommendationRepository interface {
	FindPopular(ctx context.Context, userID string, limit int, minRating float64, title string) ([]*entity.Movie, error)
	FindContentBased(ctx context.Context, userID string, limit int, minRating float64, title string) ([]*entity.Movie, error)
	FindCollaborative(ctx context.Context, userID string, limit int, minRating float64, title string) ([]*entity.Movie, error)
	FindHybrid(ctx context.Context, userID string, limit int, minRating float64, contentWeight, collaborativeWeight float64, title string) ([]*entity.Movie, error)
	FindSerendipity(ctx context.Context, userID string, limit int, minRating float64, title string) ([]*entity.Movie, error)
}
