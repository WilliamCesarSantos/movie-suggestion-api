package usecase

import (
	"context"
	"time"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/application/recommendation"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/repository"
	domainusecase "github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/usecase"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type manageUserUseCase struct {
	userRepo repository.UserRepository
	selector *recommendation.AlgorithmSelector
}

func NewManageUserUseCase(userRepo repository.UserRepository, selector *recommendation.AlgorithmSelector) domainusecase.ManageUserUseCase {
	return &manageUserUseCase{userRepo: userRepo, selector: selector}
}

func (uc *manageUserUseCase) Create(ctx context.Context, user *entity.User) error {
	logger := log.Ctx(ctx).With().Str("logger", "usecase.manage_user").Logger()
	logger.Info().Str("email", user.Email).Msg("creating user")

	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}
	user.CurrentAlgorithm = entity.AlgorithmPopular
	if err := uc.userRepo.Create(ctx, user); err != nil {
		logger.Error().Err(err).Str("userId", user.ID).Str("email", user.Email).Msg("failed to create user")
		return err
	}

	logger.Info().Str("userId", user.ID).Str("email", user.Email).Msg("user created")
	return nil
}

func (uc *manageUserUseCase) GetByID(ctx context.Context, id string) (*entity.User, error) {
	logger := log.Ctx(ctx).With().Str("logger", "usecase.manage_user").Logger()
	logger.Info().Str("userId", id).Msg("fetching user by id")

	user, err := uc.userRepo.FindByID(ctx, id)
	if err != nil {
		logger.Error().Err(err).Str("userId", id).Msg("failed to fetch user")
		return nil, err
	}

	logger.Info().Str("userId", id).Msg("user fetched")
	return user, nil
}

func (uc *manageUserUseCase) RecordWatched(ctx context.Context, userID, movieID string, userRating float64, reaction string) (*entity.User, error) {
	logger := log.Ctx(ctx).With().Str("logger", "usecase.manage_user").Logger()
	logger.Info().Str("userId", userID).Str("movieId", movieID).Float64("userRating", userRating).Str("reaction", reaction).Msg("recording watched movie")

	if err := uc.userRepo.RecordWatched(ctx, userID, movieID, userRating, reaction); err != nil {
		logger.Error().Err(err).Str("userId", userID).Str("movieId", movieID).Msg("failed to record watched movie")
		return nil, err
	}
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		logger.Error().Err(err).Str("userId", userID).Msg("failed to reload user after watched movie")
		return nil, err
	}
	newAlgo := uc.selector.Select(*user)
	if newAlgo != user.CurrentAlgorithm {
		user.CurrentAlgorithm = newAlgo
		if err := uc.userRepo.UpdateProfile(ctx, user); err != nil {
			logger.Error().Err(err).Str("userId", userID).Str("currentAlgorithm", string(newAlgo)).Msg("failed to update user algorithm")
			return nil, err
		}
		logger.Info().Str("userId", userID).Str("currentAlgorithm", string(newAlgo)).Msg("user algorithm updated")
	}
	logger.Info().Str("userId", userID).Str("movieId", movieID).Msg("watched movie recorded")
	return user, nil
}
