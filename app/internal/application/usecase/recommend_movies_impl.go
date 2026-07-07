package usecase

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/config"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/application/recommendation"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/repository"
	domainusecase "github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/usecase"
	"github.com/rs/zerolog/log"
)

type recommendMoviesUseCase struct {
	userRepo           repository.UserRepository
	recommendationRepo repository.RecommendationRepository
	selector           *recommendation.AlgorithmSelector
	dispatcher         *recommendation.AlgorithmDispatcher
	cfg                config.RecommendationConfig
}

func NewRecommendMoviesUseCase(userRepo repository.UserRepository, recommendationRepo repository.RecommendationRepository, selector *recommendation.AlgorithmSelector, dispatcher *recommendation.AlgorithmDispatcher, cfg config.RecommendationConfig) domainusecase.RecommendMoviesUseCase {
	return &recommendMoviesUseCase{userRepo: userRepo, recommendationRepo: recommendationRepo, selector: selector, dispatcher: dispatcher, cfg: cfg}
}

func (uc *recommendMoviesUseCase) Execute(ctx context.Context, userEmail string, limit int, algorithmOverride *entity.RecommendationAlgorithm, title string) ([]*entity.Movie, error) {
	logger := log.Ctx(ctx).With().Str("logger", "usecase.recommend_movies").Logger()
	logger.Info().Str("userEmail", userEmail).Int("limit", limit).Str("title", title).Msg("recommendation request received")

	user, err := uc.userRepo.FindByEmail(ctx, userEmail)
	if err != nil {
		logger.Error().Err(err).Str("userEmail", userEmail).Msg("failed to resolve user for recommendations")
		return nil, err
	}
	algo := uc.selector.Select(*user)
	if algorithmOverride != nil {
		algo = *algorithmOverride
	}
	logger.Info().Str("userId", user.ID).Str("userEmail", user.Email).Str("selectedAlgorithm", string(algo)).Msg("recommendation algorithm selected")
	if limit <= 0 {
		limit = uc.cfg.DefaultLimit
	}
	if limit > uc.cfg.MaxLimit {
		limit = uc.cfg.MaxLimit
	}
	movies, err := uc.dispatcher.Dispatch(ctx, algo, user.ID, limit, uc.cfg, title)
	if err != nil {
		logger.Error().Err(err).Str("userId", user.ID).Str("algorithm", string(algo)).Msg("failed to dispatch recommendations")
		return nil, err
	}
	if len(movies) > 0 || algo == entity.AlgorithmPopular {
		logger.Info().Str("userId", user.ID).Str("algorithm", string(algo)).Int("count", len(movies)).Msg("recommendations resolved")
		return movies, nil
	}
	logger.Info().
		Str("selectedAlgorithm", string(algo)).
		Str("fallbackAlgorithm", string(entity.AlgorithmPopular)).
		Int("limit", limit).
		Msg("recommendation fallback applied")

	fallbackMovies, fallbackErr := uc.dispatcher.Dispatch(ctx, entity.AlgorithmPopular, user.ID, limit, uc.cfg, title)
	if fallbackErr != nil {
		logger.Error().Err(fallbackErr).Str("userId", user.ID).Msg("failed to dispatch fallback recommendations")
		return nil, fallbackErr
	}
	logger.Info().Str("userId", user.ID).Str("algorithm", string(entity.AlgorithmPopular)).Int("count", len(fallbackMovies)).Msg("fallback recommendations resolved")
	return fallbackMovies, nil
}
