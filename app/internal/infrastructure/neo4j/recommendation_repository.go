package neo4j

import (
	"context"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/repository"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/neo4j/cypher"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type recommendationRepository struct {
	driver   neo4j.DriverWithContext
	database string
}

func NewRecommendationRepository(driver neo4j.DriverWithContext, database string) repository.RecommendationRepository {
	return &recommendationRepository{driver: driver, database: database}
}

func (r *recommendationRepository) FindPopular(ctx context.Context, userID string, limit int, minRating float64, title string) ([]*entity.Movie, error) {
	result, err := neo4j.ExecuteQuery(ctx, r.driver, cypher.Popular,
		map[string]any{"userId": userID, "limit": limit, "minRating": minRating, "title": title},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.database),
	)
	if err != nil {
		return nil, err
	}
	return recordsToMovies(result.Records)
}

func (r *recommendationRepository) FindContentBased(ctx context.Context, userID string, limit int, minRating float64, title string) ([]*entity.Movie, error) {
	result, err := neo4j.ExecuteQuery(ctx, r.driver, cypher.ContentBased,
		map[string]any{"userId": userID, "limit": limit, "minRating": minRating, "title": title},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.database),
	)
	if err != nil {
		return nil, err
	}
	return recordsToMovies(result.Records)
}

func (r *recommendationRepository) FindCollaborative(ctx context.Context, userID string, limit int, minRating float64, title string) ([]*entity.Movie, error) {
	result, err := neo4j.ExecuteQuery(ctx, r.driver, cypher.Collaborative,
		map[string]any{"userId": userID, "limit": limit, "minRating": minRating, "title": title},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.database),
	)
	if err != nil {
		return nil, err
	}
	return recordsToMovies(result.Records)
}

func (r *recommendationRepository) FindHybrid(ctx context.Context, userID string, limit int, minRating float64, contentWeight, collaborativeWeight float64, title string) ([]*entity.Movie, error) {
	contentResult, err := neo4j.ExecuteQuery(ctx, r.driver, cypher.ContentBased,
		map[string]any{"userId": userID, "limit": limit, "minRating": minRating, "title": title},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.database),
	)
	if err != nil {
		return nil, err
	}
	collabResult, err := neo4j.ExecuteQuery(ctx, r.driver, cypher.Collaborative,
		map[string]any{"userId": userID, "limit": limit, "minRating": minRating, "title": title},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.database),
	)
	if err != nil {
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
	return combined, nil
}

func (r *recommendationRepository) FindSerendipity(ctx context.Context, userID string, limit int, minRating float64, title string) ([]*entity.Movie, error) {
	result, err := neo4j.ExecuteQuery(ctx, r.driver, cypher.Serendipity,
		map[string]any{"userId": userID, "limit": limit, "serendipityMinRating": minRating, "title": title},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.database),
	)
	if err != nil {
		return nil, err
	}
	return recordsToMovies(result.Records)
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
