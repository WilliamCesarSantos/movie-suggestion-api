package usecase

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/repository"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/observability"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/omdb"
	"github.com/rs/zerolog/log"
)

type ProcessMovieImportUseCase interface {
	Process(ctx context.Context, imdbID string) error
}

type processMovieImportUseCase struct {
	movieRepo  repository.MovieRepository
	omdbClient *omdb.Client
	metrics    *observability.Metrics
}

func NewProcessMovieImportUseCase(movieRepo repository.MovieRepository, omdbClient *omdb.Client, metrics *observability.Metrics) ProcessMovieImportUseCase {
	return &processMovieImportUseCase{movieRepo: movieRepo, omdbClient: omdbClient, metrics: metrics}
}

func (uc *processMovieImportUseCase) Process(ctx context.Context, imdbID string) error {
	logger := log.Ctx(ctx).With().Str("logger", "usecase.process_movie_import").Logger()
	logger.Info().Str("imdbId", imdbID).Msg("processing movie import")

	existing, err := uc.movieRepo.FindByImdbID(ctx, imdbID)
	if err == nil && existing != nil {
		uc.metrics.MovieImportTotal.WithLabelValues("skipped").Inc()
		logger.Info().Str("imdbId", imdbID).Msg("movie import skipped: already exists")
		return nil
	}
	if err != nil {
		logger.Warn().Err(err).Str("imdbId", imdbID).Msg("movie import lookup failed, continuing to OMDB fetch")
	}

	movie, err := uc.omdbClient.FetchByImdbID(ctx, imdbID)
	if err != nil {
		uc.metrics.MovieImportTotal.WithLabelValues("error").Inc()
		logger.Error().Err(err).Str("imdbId", imdbID).Msg("movie import failed: OMDB fetch error")
		return err
	}

	if err := uc.movieRepo.Upsert(ctx, movie); err != nil {
		uc.metrics.MovieImportTotal.WithLabelValues("error").Inc()
		logger.Error().Err(err).Str("imdbId", imdbID).Msg("movie import failed: upsert error")
		return err
	}

	uc.metrics.MovieImportTotal.WithLabelValues("success").Inc()
	logger.Info().Str("imdbId", imdbID).Str("title", movie.Title).Msg("movie import completed")
	return nil
}
