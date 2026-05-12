package usecase

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion/internal/domain/repository"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/infrastructure/observability"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/infrastructure/omdb"
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
	existing, err := uc.movieRepo.FindByImdbID(ctx, imdbID)
	if err == nil && existing != nil {
		uc.metrics.MovieImportTotal.WithLabelValues("skipped").Inc()
		return nil
	}

	movie, err := uc.omdbClient.FetchByImdbID(ctx, imdbID)
	if err != nil {
		uc.metrics.MovieImportTotal.WithLabelValues("error").Inc()
		return err
	}

	if err := uc.movieRepo.Upsert(ctx, movie); err != nil {
		uc.metrics.MovieImportTotal.WithLabelValues("error").Inc()
		return err
	}

	uc.metrics.MovieImportTotal.WithLabelValues("success").Inc()
	return nil
}
