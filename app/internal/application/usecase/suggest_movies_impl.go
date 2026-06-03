package usecase

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/config"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/application/suggestion"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/repository"
	domainusecase "github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/usecase"
)

type suggestMoviesUseCase struct {
	userRepo       repository.UserRepository
	suggestionRepo repository.SuggestionRepository
	selector       *suggestion.AlgorithmSelector
	dispatcher     *suggestion.AlgorithmDispatcher
	cfg            config.SuggestionConfig
}

func NewSuggestMoviesUseCase(userRepo repository.UserRepository, suggestionRepo repository.SuggestionRepository, selector *suggestion.AlgorithmSelector, dispatcher *suggestion.AlgorithmDispatcher, cfg config.SuggestionConfig) domainusecase.SuggestMoviesUseCase {
	return &suggestMoviesUseCase{userRepo: userRepo, suggestionRepo: suggestionRepo, selector: selector, dispatcher: dispatcher, cfg: cfg}
}

func (uc *suggestMoviesUseCase) Execute(ctx context.Context, userID string, limit int, algorithmOverride *entity.SuggestionAlgorithm) ([]*entity.Movie, error) {
	user, err := uc.userRepo.FindByID(ctx, userID)
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
	return uc.dispatcher.Dispatch(ctx, algo, userID, limit, uc.cfg)
}
