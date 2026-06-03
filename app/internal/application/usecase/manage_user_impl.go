package usecase

import (
	"context"
	"time"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/application/suggestion"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/repository"
	domainusecase "github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/usecase"
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
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}
	user.CurrentAlgorithm = entity.AlgorithmPopular
	return uc.userRepo.Create(ctx, user)
}

func (uc *manageUserUseCase) GetByID(ctx context.Context, id string) (*entity.User, error) {
	return uc.userRepo.FindByID(ctx, id)
}

func (uc *manageUserUseCase) RecordWatched(ctx context.Context, userID, movieID string, userRating float64, reaction string) (*entity.User, error) {
	if err := uc.userRepo.RecordWatched(ctx, userID, movieID, userRating, reaction); err != nil {
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
