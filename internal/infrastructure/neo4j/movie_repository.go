package neo4j

import (
	"context"
	"fmt"

	"github.com/WilliamCesarSantos/movie-suggestion/internal/domain/entity"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/domain/repository"
	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type movieRepository struct {
	driver   neo4j.DriverWithContext
	database string
}

func NewMovieRepository(driver neo4j.DriverWithContext, database string) repository.MovieRepository {
	return &movieRepository{driver: driver, database: database}
}

func (r *movieRepository) FindByID(ctx context.Context, id string) (*entity.Movie, error) {
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		"MATCH (m:Movie {id: $id}) RETURN m",
		map[string]any{"id": id},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.database),
	)
	if err != nil {
		return nil, err
	}
	if len(result.Records) == 0 {
		return nil, entity.ErrMovieNotFound
	}
	return recordToMovie(result.Records[0])
}

func (r *movieRepository) FindByImdbID(ctx context.Context, imdbID string) (*entity.Movie, error) {
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		"MATCH (m:Movie {imdbId: $imdbId}) RETURN m",
		map[string]any{"imdbId": imdbID},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.database),
	)
	if err != nil {
		return nil, err
	}
	if len(result.Records) == 0 {
		return nil, entity.ErrMovieNotFound
	}
	return recordToMovie(result.Records[0])
}

func (r *movieRepository) Upsert(ctx context.Context, movie *entity.Movie) error {
	if movie.ID == "" {
		movie.ID = uuid.New().String()
	}
	genres := make([]string, len(movie.Genres))
	for i, g := range movie.Genres {
		genres[i] = g.Name
	}
	actors := make([]map[string]any, len(movie.Actors))
	for i, a := range movie.Actors {
		actors[i] = map[string]any{"name": a.Name, "imdbId": a.ImdbID}
	}
	directors := make([]map[string]any, len(movie.Directors))
	for i, d := range movie.Directors {
		directors[i] = map[string]any{"name": d.Name, "imdbId": d.ImdbID}
	}

	query := `
MERGE (m:Movie {imdbId: $imdbId})
SET m.id = $id, m.title = $title, m.year = $year, m.plot = $plot,
    m.runtime = $runtime, m.poster = $poster, m.imdbRating = $imdbRating
WITH m
FOREACH (g IN $genres | MERGE (genre:Genre {name: g}) MERGE (m)-[:HAS_GENRE]->(genre))
WITH m
FOREACH (a IN $actors | MERGE (actor:Actor {imdbId: a.imdbId}) SET actor.name = a.name MERGE (m)-[:HAS_ACTOR]->(actor))
WITH m
FOREACH (d IN $directors | MERGE (dir:Director {imdbId: d.imdbId}) SET dir.name = d.name MERGE (m)-[:DIRECTED_BY]->(dir))
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
	return err
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
