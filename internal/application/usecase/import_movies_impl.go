package usecase

import (
	"context"

	domainusecase "github.com/WilliamCesarSantos/movie-suggestion/internal/domain/usecase"
	"github.com/rs/zerolog/log"
)

type importMoviesUseCase struct {
	searcher  domainusecase.OmdbSearcher
	publisher domainusecase.MovieImportPublisher
}

func NewImportMoviesUseCase(searcher domainusecase.OmdbSearcher, publisher domainusecase.MovieImportPublisher) domainusecase.ImportMoviesUseCase {
	return &importMoviesUseCase{searcher: searcher, publisher: publisher}
}

func (uc *importMoviesUseCase) Execute(ctx context.Context, searchTerms []string, maxPages int) error {
	go func() {
		for _, term := range searchTerms {
			for page := 1; page <= maxPages; page++ {
				results, err := uc.searcher.Search(context.Background(), term, page)
				if err != nil {
					log.Error().Err(err).Str("term", term).Int("page", page).Msg("OMDB search error")
					continue
				}
				for _, r := range results {
					if err := uc.publisher.Publish(context.Background(), r.ImdbID); err != nil {
						log.Error().Err(err).Str("imdbId", r.ImdbID).Msg("failed to publish import message")
					}
				}
			}
		}
	}()
	return nil
}
