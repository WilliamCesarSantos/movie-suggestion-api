package usecase

import "context"

type PatchUserInput struct {
	TargetUserID string
	CallerUserID string
	Name         *string
	Password     *string
	Roles        *[]string
}

type PatchUserOutput struct {
	ID        string
	Name      string
	Email     string
	Roles     []string
	CreatedAt string
}

type PatchUserUseCase interface {
	Execute(ctx context.Context, input PatchUserInput) (*PatchUserOutput, error)
}
