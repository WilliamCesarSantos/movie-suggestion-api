package usecase

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion/internal/domain/entity"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/domain/repository"
	domainusecase "github.com/WilliamCesarSantos/movie-suggestion/internal/domain/usecase"
)

type getMovieUseCase struct {
	movieRepo repository.MovieRepository
}

func NewGetMovieUseCase(movieRepo repository.MovieRepository) domainusecase.GetMovieUseCase {
	return &getMovieUseCase{movieRepo: movieRepo}
}

func (uc *getMovieUseCase) GetByID(ctx context.Context, id string) (*entity.Movie, error) {
	return uc.movieRepo.FindByID(ctx, id)
}
