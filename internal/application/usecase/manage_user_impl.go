package usecase

import (
	"context"
	"time"

	"github.com/WilliamCesarSantos/movie-suggestion/internal/application/suggestion"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/domain/entity"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/domain/repository"
	domainusecase "github.com/WilliamCesarSantos/movie-suggestion/internal/domain/usecase"
	"github.com/google/uuid"
)

type manageUserUseCase struct {
	userRepo repository.UserRepository
	selector *suggestion.AlgorithmSelector
}

func NewManageUserUseCase(userRepo repository.UserRepository, selector *suggestion.AlgorithmSelector) domainusecase.ManageUserUseCase {
	return &manageUserUseCase{userRepo: userRepo, selector: selector}
}

func (uc *manageUserUseCase) Create(ctx context.Context, user *entity.User) error {
	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	user.CreatedAt = time.Now()
	user.CurrentAlgorithm = entity.AlgorithmPopular
	return uc.userRepo.Create(ctx, user)
}

func (uc *manageUserUseCase) GetByID(ctx context.Context, id string) (*entity.User, error) {
	return uc.userRepo.FindByID(ctx, id)
}

func (uc *manageUserUseCase) RecordWatched(ctx context.Context, userID, movieID string, rating float64) (*entity.User, error) {
	if err := uc.userRepo.RecordWatched(ctx, userID, movieID, rating); err != nil {
		return nil, err
	}
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	newAlgo := uc.selector.Select(*user)
	if newAlgo != user.CurrentAlgorithm {
		user.CurrentAlgorithm = newAlgo
		if err := uc.userRepo.UpdateProfile(ctx, user); err != nil {
			return nil, err
		}
	}
	return user, nil
}

func (uc *manageUserUseCase) RecordLiked(ctx context.Context, userID, movieID string, suggestionAlgorithmUsed entity.SuggestionAlgorithm) (*entity.User, error) {
	if err := uc.userRepo.RecordLiked(ctx, userID, movieID); err != nil {
		return nil, err
	}
	return uc.userRepo.FindByID(ctx, userID)
}

func (uc *manageUserUseCase) RecordDisliked(ctx context.Context, userID, movieID string) (*entity.User, error) {
	if err := uc.userRepo.RecordDisliked(ctx, userID, movieID); err != nil {
		return nil, err
	}
	return uc.userRepo.FindByID(ctx, userID)
}
