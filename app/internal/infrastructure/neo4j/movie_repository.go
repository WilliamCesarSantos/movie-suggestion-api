package neo4j

import (
	"context"
	"fmt"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/repository"
	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rs/zerolog/log"
)

type movieRepository struct {
	driver   neo4j.DriverWithContext
	database string
}

func NewMovieRepository(driver neo4j.DriverWithContext, database string) repository.MovieRepository {
	return &movieRepository{driver: driver, database: database}
}

func (r *movieRepository) FindByID(ctx context.Context, id string) (*entity.Movie, error) {
	logger := log.Ctx(ctx).With().Str("logger", "repo.neo4j.movie").Logger()
	logger.Info().Str("movieId", id).Msg("finding movie by id")

	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (m:Movie {id: $id})
		OPTIONAL MATCH (m)-[:HAS_GENRE]->(g:Genre)
		OPTIONAL MATCH (m)-[:HAS_ACTOR]->(a:Actor)
		OPTIONAL MATCH (m)-[:DIRECTED_BY]->(d:Director)
		RETURN m,
		       collect(DISTINCT g.name) AS genres,
		       collect(DISTINCT a.name) AS actors,
		       collect(DISTINCT d.name) AS directors`,
		map[string]any{"id": id},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.database),
	)
	if err != nil {
		logger.Error().Err(err).Str("movieId", id).Msg("failed to find movie by id")
		return nil, err
	}
	if len(result.Records) == 0 {
		logger.Warn().Str("movieId", id).Msg("movie not found by id")
		return nil, entity.ErrMovieNotFound
	}
	movie, err := recordToMovieFull(result.Records[0])
	if err != nil {
		logger.Error().Err(err).Str("movieId", id).Msg("failed to map movie by id")
		return nil, err
	}
	logger.Info().Str("movieId", id).Msg("movie found by id")
	return movie, nil
}

func (r *movieRepository) FindByImdbID(ctx context.Context, imdbID string) (*entity.Movie, error) {
	logger := log.Ctx(ctx).With().Str("logger", "repo.neo4j.movie").Logger()
	logger.Info().Str("imdbId", imdbID).Msg("finding movie by imdbId")

	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		"MATCH (m:Movie {imdbId: $imdbId}) RETURN m",
		map[string]any{"imdbId": imdbID},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.database),
	)
	if err != nil {
		logger.Error().Err(err).Str("imdbId", imdbID).Msg("failed to find movie by imdbId")
		return nil, err
	}
	if len(result.Records) == 0 {
		logger.Warn().Str("imdbId", imdbID).Msg("movie not found by imdbId")
		return nil, entity.ErrMovieNotFound
	}
	movie, err := recordToMovie(result.Records[0])
	if err != nil {
		logger.Error().Err(err).Str("imdbId", imdbID).Msg("failed to map movie by imdbId")
		return nil, err
	}
	logger.Info().Str("imdbId", imdbID).Msg("movie found by imdbId")
	return movie, nil
}

func (r *movieRepository) Upsert(ctx context.Context, movie *entity.Movie) error {
	logger := log.Ctx(ctx).With().Str("logger", "repo.neo4j.movie").Logger()
	logger.Info().Str("movieId", movie.ID).Str("imdbId", movie.ImdbID).Msg("upserting movie")

	if movie.ID == "" {
		movie.ID = uuid.New().String()
	}
	genres := make([]string, len(movie.Genres))
	for i, g := range movie.Genres {
		genres[i] = g.Name
	}
	actors := make([]map[string]any, len(movie.Actors))
	for i, a := range movie.Actors {
		actors[i] = map[string]any{"name": a.Name}
	}
	directors := make([]map[string]any, len(movie.Directors))
	for i, d := range movie.Directors {
		directors[i] = map[string]any{"name": d.Name}
	}

	query := `
MERGE (m:Movie {imdbId: $imdbId})
ON CREATE SET m.id = $id, m.createdAt = datetime()
SET m.title = $title, m.year = $year, m.plot = $plot,
    m.runtime = $runtime, m.poster = $poster, m.imdbRating = $imdbRating
WITH m
FOREACH (g IN $genres | MERGE (genre:Genre {name: g}) MERGE (m)-[:HAS_GENRE]->(genre))
WITH m
FOREACH (a IN $actors | MERGE (actor:Actor {name: a.name}) MERGE (m)-[:HAS_ACTOR]->(actor))
WITH m
FOREACH (d IN $directors | MERGE (dir:Director {name: d.name}) MERGE (m)-[:DIRECTED_BY]->(dir))
`
	_, err := neo4j.ExecuteQuery(ctx, r.driver, query,
		map[string]any{
			"imdbId":     movie.ImdbID,
			"id":         movie.ID,
			"title":      movie.Title,
			"year":       movie.Year,
			"plot":       movie.Plot,
			"runtime":    movie.Runtime,
			"poster":     movie.Poster,
			"imdbRating": movie.ImdbRating,
			"genres":     genres,
			"actors":     actors,
			"directors":  directors,
		},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.database),
	)
	if err != nil {
		logger.Error().Err(err).Str("movieId", movie.ID).Str("imdbId", movie.ImdbID).Msg("failed to upsert movie")
		return err
	}
	logger.Info().Str("movieId", movie.ID).Str("imdbId", movie.ImdbID).Msg("movie upserted")
	return nil
}

func recordToMovie(record *neo4j.Record) (*entity.Movie, error) {
	node, ok := record.Values[0].(neo4j.Node)
	if !ok {
		return nil, fmt.Errorf("expected neo4j.Node")
	}
	m := &entity.Movie{}
	if v, ok := node.Props["id"]; ok {
		m.ID, _ = v.(string)
	}
	if v, ok := node.Props["title"]; ok {
		m.Title, _ = v.(string)
	}
	if v, ok := node.Props["year"]; ok {
		m.Year, _ = v.(string)
	}
	if v, ok := node.Props["plot"]; ok {
		m.Plot, _ = v.(string)
	}
	if v, ok := node.Props["runtime"]; ok {
		m.Runtime, _ = v.(string)
	}
	if v, ok := node.Props["poster"]; ok {
		m.Poster, _ = v.(string)
	}
	if v, ok := node.Props["imdbRating"]; ok {
		m.ImdbRating, _ = v.(float64)
	}
	if v, ok := node.Props["imdbId"]; ok {
		m.ImdbID, _ = v.(string)
	}
	return m, nil
}

func recordToMovieFull(record *neo4j.Record) (*entity.Movie, error) {
	m, err := recordToMovie(record)
	if err != nil {
		return nil, err
	}
	if rawGenres, ok := record.Values[1].([]any); ok {
		m.Genres = make([]entity.Genre, 0, len(rawGenres))
		for _, v := range rawGenres {
			if name, ok := v.(string); ok && name != "" {
				m.Genres = append(m.Genres, entity.Genre{Name: name})
			}
		}
	}
	if rawActors, ok := record.Values[2].([]any); ok {
		m.Actors = make([]entity.Actor, 0, len(rawActors))
		for _, v := range rawActors {
			if name, ok := v.(string); ok && name != "" {
				m.Actors = append(m.Actors, entity.Actor{Name: name})
			}
		}
	}
	if rawDirs, ok := record.Values[3].([]any); ok {
		m.Directors = make([]entity.Director, 0, len(rawDirs))
		for _, v := range rawDirs {
			if name, ok := v.(string); ok && name != "" {
				m.Directors = append(m.Directors, entity.Director{Name: name})
			}
		}
	}
	return m, nil
}
