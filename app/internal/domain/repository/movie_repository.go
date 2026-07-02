package repository

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
)

type MovieRepository interface {
	FindByID(ctx context.Context, id string) (*entity.Movie, error)
	FindByImdbID(ctx context.Context, imdbID string) (*entity.Movie, error)
	Upsert(ctx context.Context, movie *entity.Movie) error
}
