package usecase

import (
	"context"
	"time"
)

type LoginResult struct {
	Token     string
	Email     string
	Roles     []string
	ExpiresAt time.Time
}

type LoginUseCase interface {
	Execute(ctx context.Context, email, password string) (*LoginResult, error)
}
