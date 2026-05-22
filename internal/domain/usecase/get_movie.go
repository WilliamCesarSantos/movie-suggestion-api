package usecase

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion/internal/domain/entity"
)

type GetMovieUseCase interface {
	GetByID(ctx context.Context, id string) (*entity.Movie, error)
}
