package usecase

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion/internal/domain/entity"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/domain/repository"
	domainusecase "github.com/WilliamCesarSantos/movie-suggestion/internal/domain/usecase"
)

type updateUserProfileUseCase struct {
	userRepo repository.UserRepository
}

func NewUpdateUserProfileUseCase(userRepo repository.UserRepository) domainusecase.UpdateUserProfileUseCase {
	return &updateUserProfileUseCase{userRepo: userRepo}
}

func (uc *updateUserProfileUseCase) Execute(ctx context.Context, user *entity.User) error {
	return uc.userRepo.UpdateProfile(ctx, user)
}
