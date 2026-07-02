package usecase

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/repository"
	domainusecase "github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/usecase"
)

type listUsersUseCase struct {
	authUserRepo repository.AuthUserRepository
}

func NewListUsersUseCase(authUserRepo repository.AuthUserRepository) domainusecase.ListUsersUseCase {
	return &listUsersUseCase{authUserRepo: authUserRepo}
}

func (uc *listUsersUseCase) Execute(ctx context.Context, callerEmail string, callerHasWrite bool, input domainusecase.ListUsersInput) (*domainusecase.ListUsersOutput, error) {
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

	return &domainusecase.ListUsersOutput{
		Users:    users,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
