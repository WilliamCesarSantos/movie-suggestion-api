package usecase

import (
	"context"
	"errors"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/repository"
	domainusecase "github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/usecase"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/auth"
	"github.com/rs/zerolog/log"
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
	logger := log.Ctx(ctx).With().Str("logger", "usecase.login").Logger()
	logger.Info().Str("email", email).Msg("login attempt")

	user, err := uc.authUserRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, entity.ErrAuthUserNotFound) {
			logger.Warn().Str("email", email).Msg("login failed: user not found")
			return nil, entity.ErrAuthUserNotFound
		}
		logger.Error().Err(err).Str("email", email).Msg("login failed: auth user lookup error")
		return nil, err
	}
	ok, err := uc.passwordService.Verify(password, user.Password)
	if err != nil || !ok {
		logger.Warn().Str("email", email).Msg("login failed: invalid credentials")
		return nil, entity.ErrAuthUserNotFound
	}
	token, expiresAt, err := uc.jwtService.Generate(user.ID, user.Email, user.Roles)
	if err != nil {
		logger.Error().Err(err).Str("userId", user.ID).Str("email", user.Email).Msg("login failed: token generation error")
		return nil, err
	}
	logger.Info().Str("userId", user.ID).Str("email", user.Email).Msg("login succeeded")
	return &domainusecase.LoginResult{
		Token:     token,
		Email:     user.Email,
		Roles:     user.Roles,
		ExpiresAt: expiresAt,
	}, nil
}
