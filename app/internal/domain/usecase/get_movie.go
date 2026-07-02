package usecase

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
)

type GetMovieUseCase interface {
	GetByID(ctx context.Context, id string) (*entity.Movie, error)
}
