package usecase

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
)

type ListUsersInput struct {
	Email    string
	Name     string
	Page     int
	PageSize int
}

type ListUsersOutput struct {
	Users    []*entity.AuthUser
	Total    int
	Page     int
	PageSize int
}

type ListUsersUseCase interface {
	Execute(ctx context.Context, callerEmail string, callerHasWrite bool, input ListUsersInput) (*ListUsersOutput, error)
}
