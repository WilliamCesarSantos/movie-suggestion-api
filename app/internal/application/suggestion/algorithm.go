package suggestion

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/config"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/repository"
)

type AlgorithmDispatcher struct {
	repo repository.SuggestionRepository
}

func NewAlgorithmDispatcher(repo repository.SuggestionRepository) *AlgorithmDispatcher {
	return &AlgorithmDispatcher{repo: repo}
}

func (d *AlgorithmDispatcher) Dispatch(ctx context.Context, algo entity.SuggestionAlgorithm, userID string, limit int, cfg config.SuggestionConfig) ([]*entity.Movie, error) {
	switch algo {
	case entity.AlgorithmPopular:
		return d.repo.FindPopular(ctx, userID, limit, cfg.MinImdbRating)
	case entity.AlgorithmContentBased:
		return d.repo.FindContentBased(ctx, userID, limit, cfg.MinImdbRating)
	case entity.AlgorithmCollaborative:
		return d.repo.FindCollaborative(ctx, userID, limit, cfg.MinImdbRating)
	case entity.AlgorithmHybrid:
		return d.repo.FindHybrid(ctx, userID, limit, cfg.MinImdbRating, cfg.HybridContentWeight, cfg.HybridCollaborativeWeight)
	case entity.AlgorithmSerendipity:
		return d.repo.FindSerendipity(ctx, userID, limit, cfg.SerendipityMinRating)
	default:
		return nil, entity.ErrAlgorithmNotFound
	}
}
