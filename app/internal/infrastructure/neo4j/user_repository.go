package neo4j

import (
	"context"
	"fmt"
	"time"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/repository"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rs/zerolog/log"
)

type userRepository struct {
	driver   neo4j.DriverWithContext
	database string
}

func NewUserRepository(driver neo4j.DriverWithContext, database string) repository.UserRepository {
	return &userRepository{driver: driver, database: database}
}

func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
	logger := log.Ctx(ctx).With().Str("logger", "repo.neo4j.user").Logger()
	logger.Info().Str("userId", user.ID).Str("email", user.Email).Msg("creating graph user")

	_, err := neo4j.ExecuteQuery(ctx, r.driver,
		`CREATE (u:User {id: $id, name: $name, email: $email, createdAt: $createdAt,
         currentAlgorithm: $currentAlgorithm, watchCount: 0, likeCount: 0, dislikeCount: 0})`,
		map[string]any{
			"id":               user.ID,
			"name":             user.Name,
			"email":            user.Email,
			"createdAt":        user.CreatedAt.Format(time.RFC3339),
			"currentAlgorithm": string(user.CurrentAlgorithm),
		},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.database),
	)
	if err != nil {
		logger.Error().Err(err).Str("userId", user.ID).Str("email", user.Email).Msg("failed to create graph user")
		return err
	}
	logger.Info().Str("userId", user.ID).Str("email", user.Email).Msg("graph user created")
	return err
}

func (r *userRepository) FindByID(ctx context.Context, id string) (*entity.User, error) {
	logger := log.Ctx(ctx).With().Str("logger", "repo.neo4j.user").Logger()
	logger.Info().Str("userId", id).Msg("finding graph user by id")

	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		"MATCH (u:User {id: $id}) RETURN u",
		map[string]any{"id": id},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.database),
	)
	if err != nil {
		logger.Error().Err(err).Str("userId", id).Msg("failed to find graph user by id")
		return nil, err
	}
	if len(result.Records) == 0 {
		logger.Warn().Str("userId", id).Msg("graph user not found by id")
		return nil, entity.ErrUserNotFound
	}
	user, err := recordToUser(result.Records[0])
	if err != nil {
		logger.Error().Err(err).Str("userId", id).Msg("failed to map graph user by id")
		return nil, err
	}
	logger.Info().Str("userId", id).Msg("graph user found by id")
	return user, nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	logger := log.Ctx(ctx).With().Str("logger", "repo.neo4j.user").Logger()
	logger.Info().Str("email", email).Msg("finding graph user by email")

	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		"MATCH (u:User {email: $email}) RETURN u",
		map[string]any{"email": email},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.database),
	)
	if err != nil {
		logger.Error().Err(err).Str("email", email).Msg("failed to find graph user by email")
		return nil, err
	}
	if len(result.Records) == 0 {
		logger.Warn().Str("email", email).Msg("graph user not found by email")
		return nil, entity.ErrUserNotFound
	}
	user, err := recordToUser(result.Records[0])
	if err != nil {
		logger.Error().Err(err).Str("email", email).Msg("failed to map graph user by email")
		return nil, err
	}
	logger.Info().Str("email", email).Msg("graph user found by email")
	return user, nil
}

func (r *userRepository) UpdateProfile(ctx context.Context, user *entity.User) error {
	logger := log.Ctx(ctx).With().Str("logger", "repo.neo4j.user").Logger()
	logger.Info().Str("userId", user.ID).Str("email", user.Email).Msg("updating graph user profile")

	_, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (u:User {id: $id})
         SET u.name = $name, u.email = $email, u.currentAlgorithm = $currentAlgorithm`,
		map[string]any{
			"id":               user.ID,
			"name":             user.Name,
			"email":            user.Email,
			"currentAlgorithm": string(user.CurrentAlgorithm),
		},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.database),
	)
	if err != nil {
		logger.Error().Err(err).Str("userId", user.ID).Str("email", user.Email).Msg("failed to update graph user profile")
		return err
	}
	logger.Info().Str("userId", user.ID).Str("email", user.Email).Msg("graph user profile updated")
	return err
}

func (r *userRepository) RecordWatched(ctx context.Context, userID, movieID string, userRating float64, reaction string) error {
	logger := log.Ctx(ctx).With().Str("logger", "repo.neo4j.user").Logger()
	logger.Info().Str("userId", userID).Str("movieId", movieID).Float64("userRating", userRating).Str("reaction", reaction).Msg("recording watched relation")

	existence, err := neo4j.ExecuteQuery(ctx, r.driver,
		`OPTIONAL MATCH (u:User {id: $userId})
         OPTIONAL MATCH (m:Movie {id: $movieId})
         RETURN u IS NOT NULL AS userExists, m IS NOT NULL AS movieExists`,
		map[string]any{
			"userId":  userID,
			"movieId": movieID,
		},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.database),
	)
	if err != nil {
		logger.Error().Err(err).Str("userId", userID).Str("movieId", movieID).Msg("failed to validate watched relation")
		return err
	}
	if len(existence.Records) == 0 {
		logger.Warn().Str("userId", userID).Str("movieId", movieID).Msg("watch validation returned no records")
		return entity.ErrUserNotFound
	}

	userExists, _ := existence.Records[0].Get("userExists")
	movieExists, _ := existence.Records[0].Get("movieExists")
	if ok, _ := userExists.(bool); !ok {
		logger.Warn().Str("userId", userID).Msg("user not found while recording watched relation")
		return entity.ErrUserNotFound
	}
	if ok, _ := movieExists.(bool); !ok {
		logger.Warn().Str("movieId", movieID).Msg("movie not found while recording watched relation")
		return entity.ErrMovieNotFound
	}

	_, err = neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (u:User {id: $userId}), (m:Movie {id: $movieId})
         MERGE (u)-[w:WATCHED]->(m)
         SET w.watchedAt = $watchedAt, w.userRating = $userRating, w.reaction = $reaction
         SET u.watchCount = u.watchCount + 1
         FOREACH (_ IN CASE WHEN $reaction = 'liked' THEN [1] ELSE [] END |
           MERGE (u)-[:LIKED]->(m)
         )`,
		map[string]any{
			"userId":     userID,
			"movieId":    movieID,
			"watchedAt":  time.Now().Format(time.RFC3339),
			"userRating": userRating,
			"reaction":   reaction,
		},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.database),
	)
	if err != nil {
		logger.Error().Err(err).Str("userId", userID).Str("movieId", movieID).Msg("failed to record watched relation")
		return err
	}
	logger.Info().Str("userId", userID).Str("movieId", movieID).Msg("watched relation recorded")
	return err
}

func recordToUser(record *neo4j.Record) (*entity.User, error) {
	node, ok := record.Values[0].(neo4j.Node)
	if !ok {
		return nil, fmt.Errorf("expected neo4j.Node")
	}
	u := &entity.User{}
	if v, ok := node.Props["id"]; ok {
		u.ID, _ = v.(string)
	}
	if v, ok := node.Props["name"]; ok {
		u.Name, _ = v.(string)
	}
	if v, ok := node.Props["email"]; ok {
		u.Email, _ = v.(string)
	}
	if v, ok := node.Props["currentAlgorithm"]; ok {
		u.CurrentAlgorithm = entity.RecommendationAlgorithm(v.(string))
	}
	if v, ok := node.Props["watchCount"]; ok {
		if wc, ok2 := v.(int64); ok2 {
			u.WatchCount = int(wc)
		}
	}
	if v, ok := node.Props["likeCount"]; ok {
		if lc, ok2 := v.(int64); ok2 {
			u.LikeCount = int(lc)
		}
	}
	if v, ok := node.Props["dislikeCount"]; ok {
		if dc, ok2 := v.(int64); ok2 {
			u.DislikeCount = int(dc)
		}
	}
	if v, ok := node.Props["createdAt"]; ok {
		if s, ok2 := v.(string); ok2 {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				u.CreatedAt = t
			}
		}
	}
	return u, nil
}
