package usecase

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/repository"
	domainusecase "github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/usecase"
	"github.com/rs/zerolog/log"
)

type listUsersUseCase struct {
	authUserRepo repository.AuthUserRepository
}

func NewListUsersUseCase(authUserRepo repository.AuthUserRepository) domainusecase.ListUsersUseCase {
	return &listUsersUseCase{authUserRepo: authUserRepo}
}

func (uc *listUsersUseCase) Execute(ctx context.Context, callerEmail string, callerHasWrite bool, input domainusecase.ListUsersInput) (*domainusecase.ListUsersOutput, error) {
	logger := log.Ctx(ctx).With().Str("logger", "usecase.list_users").Logger()
	logger.Info().Str("callerEmail", callerEmail).Bool("callerHasWrite", callerHasWrite).Int("page", input.Page).Int("pageSize", input.PageSize).Msg("listing users")

	filters := repository.AuthUserFilters{
		Name:     input.Name,
		Page:     input.Page,
		PageSize: input.PageSize,
	}

	if !callerHasWrite {
		// users:read only — restrict to own record
		filters.Email = callerEmail
	} else if input.Email != "" {
		filters.Email = input.Email
	}

	users, total, err := uc.authUserRepo.List(ctx, filters)
	if err != nil {
		logger.Error().Err(err).Interface("filters", filters).Msg("failed to list users")
		return nil, err
	}

	page := input.Page
	if page < 1 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	output := &domainusecase.ListUsersOutput{
		Users:    users,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}
	logger.Info().Int("total", total).Int("page", page).Int("pageSize", pageSize).Int("returned", len(users)).Msg("users listed")
	return output, nil
}
