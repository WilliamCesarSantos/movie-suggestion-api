package usecase

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/repository"
	domainusecase "github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/usecase"
	"github.com/rs/zerolog/log"
)

type getMovieUseCase struct {
	movieRepo repository.MovieRepository
}

func NewGetMovieUseCase(movieRepo repository.MovieRepository) domainusecase.GetMovieUseCase {
	return &getMovieUseCase{movieRepo: movieRepo}
}

func (uc *getMovieUseCase) GetByID(ctx context.Context, id string) (*entity.Movie, error) {
	logger := log.Ctx(ctx).With().Str("logger", "usecase.get_movie").Logger()
	logger.Info().Str("movieId", id).Msg("fetching movie by id")

	movie, err := uc.movieRepo.FindByID(ctx, id)
	if err != nil {
		logger.Error().Err(err).Str("movieId", id).Msg("failed to fetch movie")
		return nil, err
	}

	logger.Info().Str("movieId", id).Msg("movie fetched")
	return movie, nil
}
