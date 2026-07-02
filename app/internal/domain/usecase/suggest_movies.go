package usecase

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
)

type SuggestMoviesUseCase interface {
	Execute(ctx context.Context, userEmail string, limit int, algorithmOverride *entity.SuggestionAlgorithm) ([]*entity.Movie, error)
}
