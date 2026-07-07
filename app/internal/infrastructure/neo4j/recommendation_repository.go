package neo4j

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/repository"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/neo4j/cypher"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rs/zerolog/log"
)

type recommendationRepository struct {
	driver   neo4j.DriverWithContext
	database string
}

func NewRecommendationRepository(driver neo4j.DriverWithContext, database string) repository.RecommendationRepository {
	return &recommendationRepository{driver: driver, database: database}
}

func (r *recommendationRepository) FindPopular(ctx context.Context, userID string, limit int, minRating float64, title string) ([]*entity.Movie, error) {
	logger := log.Ctx(ctx).With().Str("logger", "repo.neo4j.recommendation").Str("algorithm", "POPULAR").Logger()
	logger.Info().Str("userId", userID).Int("limit", limit).Float64("minRating", minRating).Str("title", title).Msg("finding popular recommendations")

	result, err := neo4j.ExecuteQuery(ctx, r.driver, cypher.Popular,
		map[string]any{"userId": userID, "limit": limit, "minRating": minRating, "title": title},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.database),
	)
	if err != nil {
		logger.Error().Err(err).Str("userId", userID).Msg("failed to find popular recommendations")
		return nil, err
	}
	movies, err := recordsToMovies(result.Records)
	if err != nil {
		logger.Error().Err(err).Str("userId", userID).Msg("failed to map popular recommendations")
		return nil, err
	}
	logger.Info().Str("userId", userID).Int("count", len(movies)).Msg("popular recommendations found")
	return movies, nil
}

func (r *recommendationRepository) FindContentBased(ctx context.Context, userID string, limit int, minRating float64, title string) ([]*entity.Movie, error) {
	logger := log.Ctx(ctx).With().Str("logger", "repo.neo4j.recommendation").Str("algorithm", "CONTENT_BASED").Logger()
	logger.Info().Str("userId", userID).Int("limit", limit).Float64("minRating", minRating).Str("title", title).Msg("finding content-based recommendations")

	result, err := neo4j.ExecuteQuery(ctx, r.driver, cypher.ContentBased,
		map[string]any{"userId": userID, "limit": limit, "minRating": minRating, "title": title},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.database),
	)
	if err != nil {
		logger.Error().Err(err).Str("userId", userID).Msg("failed to find content-based recommendations")
		return nil, err
	}
	movies, err := recordsToMovies(result.Records)
	if err != nil {
		logger.Error().Err(err).Str("userId", userID).Msg("failed to map content-based recommendations")
		return nil, err
	}
	logger.Info().Str("userId", userID).Int("count", len(movies)).Msg("content-based recommendations found")
	return movies, nil
}

func (r *recommendationRepository) FindCollaborative(ctx context.Context, userID string, limit int, minRating float64, title string) ([]*entity.Movie, error) {
	logger := log.Ctx(ctx).With().Str("logger", "repo.neo4j.recommendation").Str("algorithm", "COLLABORATIVE").Logger()
	logger.Info().Str("userId", userID).Int("limit", limit).Float64("minRating", minRating).Str("title", title).Msg("finding collaborative recommendations")

	result, err := neo4j.ExecuteQuery(ctx, r.driver, cypher.Collaborative,
		map[string]any{"userId": userID, "limit": limit, "minRating": minRating, "title": title},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.database),
	)
	if err != nil {
		logger.Error().Err(err).Str("userId", userID).Msg("failed to find collaborative recommendations")
		return nil, err
	}
	movies, err := recordsToMovies(result.Records)
	if err != nil {
		logger.Error().Err(err).Str("userId", userID).Msg("failed to map collaborative recommendations")
		return nil, err
	}
	logger.Info().Str("userId", userID).Int("count", len(movies)).Msg("collaborative recommendations found")
	return movies, nil
}

func (r *recommendationRepository) FindHybrid(ctx context.Context, userID string, limit int, minRating float64, contentWeight, collaborativeWeight float64, title string) ([]*entity.Movie, error) {
	logger := log.Ctx(ctx).With().Str("logger", "repo.neo4j.recommendation").Str("algorithm", "HYBRID").Logger()
	logger.Info().Str("userId", userID).Int("limit", limit).Float64("minRating", minRating).Float64("contentWeight", contentWeight).Float64("collaborativeWeight", collaborativeWeight).Str("title", title).Msg("finding hybrid recommendations")

	contentResult, err := neo4j.ExecuteQuery(ctx, r.driver, cypher.ContentBased,
		map[string]any{"userId": userID, "limit": limit, "minRating": minRating, "title": title},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.database),
	)
	if err != nil {
		logger.Error().Err(err).Str("userId", userID).Msg("failed to find hybrid content-based part")
		return nil, err
	}
	collabResult, err := neo4j.ExecuteQuery(ctx, r.driver, cypher.Collaborative,
		map[string]any{"userId": userID, "limit": limit, "minRating": minRating, "title": title},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.database),
	)
	if err != nil {
		logger.Error().Err(err).Str("userId", userID).Msg("failed to find hybrid collaborative part")
		return nil, err
	}

	seen := map[string]bool{}
	var combined []*entity.Movie

	contentMovies, _ := recordsToMovies(contentResult.Records)
	collabMovies, _ := recordsToMovies(collabResult.Records)

	contentCount := int(float64(limit) * contentWeight)
	collabCount := limit - contentCount

	for i, m := range contentMovies {
		if i >= contentCount {
			break
		}
		if !seen[m.ImdbID] {
			seen[m.ImdbID] = true
			combined = append(combined, m)
		}
	}
	for i, m := range collabMovies {
		if i >= collabCount || len(combined) >= limit {
			break
		}
		if !seen[m.ImdbID] {
			seen[m.ImdbID] = true
			combined = append(combined, m)
		}
	}
	logger.Info().Str("userId", userID).Int("count", len(combined)).Msg("hybrid recommendations found")
	return combined, nil
}

func (r *recommendationRepository) FindSerendipity(ctx context.Context, userID string, limit int, minRating float64, title string) ([]*entity.Movie, error) {
	logger := log.Ctx(ctx).With().Str("logger", "repo.neo4j.recommendation").Str("algorithm", "SERENDIPITY").Logger()
	logger.Info().Str("userId", userID).Int("limit", limit).Float64("minRating", minRating).Str("title", title).Msg("finding serendipity recommendations")

	result, err := neo4j.ExecuteQuery(ctx, r.driver, cypher.Serendipity,
		map[string]any{"userId": userID, "limit": limit, "serendipityMinRating": minRating, "title": title},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.database),
	)
	if err != nil {
		logger.Error().Err(err).Str("userId", userID).Msg("failed to find serendipity recommendations")
		return nil, err
	}
	movies, err := recordsToMovies(result.Records)
	if err != nil {
		logger.Error().Err(err).Str("userId", userID).Msg("failed to map serendipity recommendations")
		return nil, err
	}
	logger.Info().Str("userId", userID).Int("count", len(movies)).Msg("serendipity recommendations found")
	return movies, nil
}

func recordsToMovies(records []*neo4j.Record) ([]*entity.Movie, error) {
	movies := make([]*entity.Movie, 0, len(records))
	for _, rec := range records {
		m, err := recordToMovie(rec)
		if err != nil {
			continue
		}
		movies = append(movies, m)
	}
	return movies, nil
}
