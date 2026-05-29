package usecase

import (
	"context"
	"errors"

	"github.com/WilliamCesarSantos/movie-suggestion/app/internal/domain/entity"
	"github.com/WilliamCesarSantos/movie-suggestion/app/internal/domain/repository"
	domainusecase "github.com/WilliamCesarSantos/movie-suggestion/app/internal/domain/usecase"
	"github.com/WilliamCesarSantos/movie-suggestion/app/internal/infrastructure/auth"
)

type loginUseCase struct {
	authUserRepo    repository.AuthUserRepository
	passwordService *auth.PasswordService
	jwtService      *auth.JWTService
}

func NewLoginUseCase(authUserRepo repository.AuthUserRepository, passwordService *auth.PasswordService, jwtService *auth.JWTService) domainusecase.LoginUseCase {
	return &loginUseCase{
		authUserRepo:    authUserRepo,
		passwordService: passwordService,
		jwtService:      jwtService,
	}
}

func (uc *loginUseCase) Execute(ctx context.Context, email, password string) (*domainusecase.LoginResult, error) {
	user, err := uc.authUserRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, entity.ErrAuthUserNotFound) {
			return nil, entity.ErrAuthUserNotFound
		}
		return nil, err
	}
	ok, err := uc.passwordService.Verify(password, user.Password)
	if err != nil || !ok {
		return nil, entity.ErrAuthUserNotFound
	}
	token, expiresAt, err := uc.jwtService.Generate(user.ID, user.Email, user.Roles)
	if err != nil {
		return nil, err
	}
	return &domainusecase.LoginResult{
		Token:     token,
		Email:     user.Email,
		Roles:     user.Roles,
		ExpiresAt: expiresAt,
	}, nil
}
