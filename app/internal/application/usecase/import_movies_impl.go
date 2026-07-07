package usecase

import (
	"context"

	domainusecase "github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/usecase"
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
	bgCtx := detachContext(ctx)
	logger := log.Ctx(bgCtx).With().Str("logger", "usecase.import_movies").Logger()
	logger.Info().Int("searchTermsCount", len(searchTerms)).Int("maxPages", maxPages).Msg("movie import job queued")

	go func() {
		defer logger.Info().Msg("movie import job finished")
		for _, term := range searchTerms {
			logger.Info().Str("term", term).Int("maxPages", maxPages).Msg("processing search term")
			for page := 1; page <= maxPages; page++ {
				results, err := uc.searcher.Search(bgCtx, term, page)
				if err != nil {
					logger.Error().Err(err).Str("term", term).Int("page", page).Msg("OMDB search error")
					continue
				}
				logger.Info().Str("term", term).Int("page", page).Int("results", len(results)).Msg("OMDB search completed")
				if len(results) == 0 {
					break
				}
				for _, r := range results {
					if err := uc.publisher.Publish(bgCtx, r.ImdbID); err != nil {
						logger.Error().Err(err).Str("imdbId", r.ImdbID).Str("term", term).Int("page", page).Msg("failed to publish import message")
						continue
					}
					logger.Info().Str("imdbId", r.ImdbID).Str("term", term).Int("page", page).Msg("import message published")
				}
			}
		}
	}()
	return nil
}
