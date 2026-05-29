package usecase

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion/app/internal/domain/entity"
)

type GetMovieUseCase interface {
	GetByID(ctx context.Context, id string) (*entity.Movie, error)
	ListMovies(ctx context.Context, page, limit int) ([]*entity.Movie, int, error)
}
