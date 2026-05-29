package usecase

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion/app/internal/domain/entity"
)

type SuggestMoviesUseCase interface {
	Execute(ctx context.Context, userID string, limit int, algorithmOverride *entity.SuggestionAlgorithm) ([]*entity.Movie, error)
}
