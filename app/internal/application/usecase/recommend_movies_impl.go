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
	user, err := uc.userRepo.FindByEmail(ctx, userEmail)
	if err != nil {
		return nil, err
	}
	algo := uc.selector.Select(*user)
	if algorithmOverride != nil {
		algo = *algorithmOverride
	}
	if limit <= 0 {
		limit = uc.cfg.DefaultLimit
	}
	if limit > uc.cfg.MaxLimit {
		limit = uc.cfg.MaxLimit
	}
	movies, err := uc.dispatcher.Dispatch(ctx, algo, user.ID, limit, uc.cfg, title)
	if err != nil {
		return nil, err
	}
	if len(movies) > 0 || algo == entity.AlgorithmPopular {
		return movies, nil
	}
	log.Ctx(ctx).Info().
		Str("selectedAlgorithm", string(algo)).
		Str("fallbackAlgorithm", string(entity.AlgorithmPopular)).
		Int("limit", limit).
		Msg("recommendation fallback applied")

	return uc.dispatcher.Dispatch(ctx, entity.AlgorithmPopular, user.ID, limit, uc.cfg, title)
}
