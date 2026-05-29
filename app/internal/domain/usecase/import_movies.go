package usecase

import "context"

type SearchResult struct {
	ImdbID string
	Title  string
}

type OmdbSearcher interface {
	Search(ctx context.Context, term string, page int) ([]SearchResult, error)
}

type MovieImportPublisher interface {
	Publish(ctx context.Context, imdbID string) error
}

type ImportMoviesUseCase interface {
	Execute(ctx context.Context, searchTerms []string, maxPages int) error
}
