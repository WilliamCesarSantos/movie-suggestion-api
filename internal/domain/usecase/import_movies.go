package usecase

import "context"

type ImportMoviesUseCase interface {
	Execute(ctx context.Context, searchTerms []string, maxPages int) error
}
