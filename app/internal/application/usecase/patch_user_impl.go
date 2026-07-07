package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/repository"
	domainusecase "github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/usecase"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/auth"
	"github.com/rs/zerolog/log"
)

var allowedPatchRoles = map[string]struct{}{
	"users:read":         {},
	"users:write":        {},
	"movies:read":        {},
	"movies-watch:write": {},
	"movies:write":       {},
}

type patchUserUseCase struct {
	authUserRepo    repository.AuthUserRepository
	userRepo        repository.UserRepository
	passwordService *auth.PasswordService
}

func NewPatchUserUseCase(authUserRepo repository.AuthUserRepository, userRepo repository.UserRepository, passwordService *auth.PasswordService) domainusecase.PatchUserUseCase {
	return &patchUserUseCase{
		authUserRepo:    authUserRepo,
		userRepo:        userRepo,
		passwordService: passwordService,
	}
}

func (uc *patchUserUseCase) Execute(ctx context.Context, input domainusecase.PatchUserInput) (*domainusecase.PatchUserOutput, error) {
	logger := log.Ctx(ctx).With().Str("logger", "usecase.patch_user").Logger()
	logger.Info().Str("targetUserId", input.TargetUserID).Str("callerUserId", input.CallerUserID).Msg("patch user requested")

	if input.Name == nil && input.Password == nil && input.Roles == nil {
		logger.Warn().Str("targetUserId", input.TargetUserID).Msg("patch user rejected: empty payload")
		return nil, entity.ErrInvalidUserPatchInput
	}
	if input.TargetUserID == "" || input.CallerUserID == "" {
		logger.Warn().Msg("patch user rejected: missing user ids")
		return nil, entity.ErrInvalidUserPatchInput
	}

	isOwner := input.TargetUserID == input.CallerUserID
	if !isOwner && (input.Name != nil || input.Password != nil) {
		logger.Warn().Str("targetUserId", input.TargetUserID).Str("callerUserId", input.CallerUserID).Msg("patch user forbidden: non-owner change")
		return nil, entity.ErrUserPatchForbidden
	}

	targetUser, err := uc.authUserRepo.FindByID(ctx, input.TargetUserID)
	if err != nil {
		logger.Error().Err(err).Str("targetUserId", input.TargetUserID).Msg("failed to load target auth user")
		return nil, err
	}

	update := repository.AuthUserUpdate{}
	nameChanged := false

	if input.Name != nil {
		trimmedName := strings.TrimSpace(*input.Name)
		if trimmedName == "" {
			logger.Warn().Str("targetUserId", input.TargetUserID).Msg("patch user rejected: empty name")
			return nil, entity.ErrInvalidUserPatchInput
		}
		update.Name = &trimmedName
		targetUser.Name = trimmedName
		nameChanged = true
	}

	if input.Password != nil {
		if len(*input.Password) < 6 {
			logger.Warn().Str("targetUserId", input.TargetUserID).Msg("patch user rejected: password too short")
			return nil, entity.ErrInvalidUserPatchInput
		}
		hashedPassword, hashErr := uc.passwordService.Hash(*input.Password)
		if hashErr != nil {
			return nil, hashErr
		}
		update.Password = &hashedPassword
	}

	if input.Roles != nil {
		if !isAllowedRoles(*input.Roles) {
			logger.Warn().Str("targetUserId", input.TargetUserID).Msg("patch user rejected: invalid roles")
			return nil, entity.ErrInvalidUserPatchInput
		}
		rolesCopy := append([]string(nil), (*input.Roles)...)
		update.Roles = &rolesCopy
		targetUser.Roles = rolesCopy
	}

	if err := uc.authUserRepo.Update(ctx, input.TargetUserID, update); err != nil {
		logger.Error().Err(err).Str("targetUserId", input.TargetUserID).Msg("failed to update auth user")
		return nil, err
	}

	if nameChanged {
		if err := uc.syncGraphUserName(ctx, targetUser.ID, targetUser.Email, targetUser.Name); err != nil {
			logger.Error().Err(err).Str("targetUserId", targetUser.ID).Msg("failed to sync graph user name")
			return nil, err
		}
	}

	logger.Info().Str("targetUserId", targetUser.ID).Bool("nameChanged", nameChanged).Bool("passwordChanged", input.Password != nil).Bool("rolesChanged", input.Roles != nil).Msg("patch user completed")

	return &domainusecase.PatchUserOutput{
		ID:        targetUser.ID,
		Name:      targetUser.Name,
		Email:     targetUser.Email,
		Roles:     append([]string(nil), targetUser.Roles...),
		CreatedAt: targetUser.CreatedAt.Format(time.RFC3339),
	}, nil
}

func isAllowedRoles(roles []string) bool {
	for _, role := range roles {
		if _, ok := allowedPatchRoles[role]; !ok {
			return false
		}
	}
	return true
}

func (uc *patchUserUseCase) syncGraphUserName(ctx context.Context, targetUserID, targetUserEmail, newName string) error {
	graphUser, err := uc.userRepo.FindByID(ctx, targetUserID)
	if err != nil {
		if !errors.Is(err, entity.ErrUserNotFound) {
			return err
		}
		graphUser, err = uc.userRepo.FindByEmail(ctx, targetUserEmail)
		if err != nil {
			return err
		}
	}

	graphUser.Name = newName
	return uc.userRepo.UpdateProfile(ctx, graphUser)
}
