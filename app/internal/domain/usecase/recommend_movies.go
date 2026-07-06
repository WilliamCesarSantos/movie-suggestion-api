package usecase

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
)

type RecommendMoviesUseCase interface {
	Execute(ctx context.Context, userEmail string, limit int, algorithmOverride *entity.RecommendationAlgorithm, title string) ([]*entity.Movie, error)
}
