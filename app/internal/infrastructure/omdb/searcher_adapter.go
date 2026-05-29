package omdb

import (
	"context"

	domainusecase "github.com/WilliamCesarSantos/movie-suggestion/app/internal/domain/usecase"
)

type SearcherAdapter struct {
	client *Client
}

func NewSearcherAdapter(client *Client) *SearcherAdapter {
	return &SearcherAdapter{client: client}
}

func (a *SearcherAdapter) Search(ctx context.Context, term string, page int) ([]domainusecase.SearchResult, error) {
	results, err := a.client.Search(ctx, term, page)
	if err != nil {
		return nil, err
	}
	out := make([]domainusecase.SearchResult, len(results))
	for i, r := range results {
		out[i] = domainusecase.SearchResult{ImdbID: r.ImdbID, Title: r.Title}
	}
	return out, nil
}
