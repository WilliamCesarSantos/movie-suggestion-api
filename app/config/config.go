package config

import (
	"os"
	"strconv"

	"github.com/WilliamCesarSantos/movie-suggestion/app/internal/domain/entity"
)

type Config struct {
	Neo4j      Neo4jConfig
	OMDB       OMDBConfig
	Suggestion SuggestionConfig
	SQS        SQSConfig
	AWS        AWSConfig
	Postgres   PostgresConfig
	Auth       AuthConfig
	Otel       OtelConfig
	Server     ServerConfig
	Log        LogConfig
}

type Neo4jConfig struct {
	URI      string
	Username string
	Password string
	Database string
}

type OMDBConfig struct {
	BaseURL        string
	APIKey         string
	TimeoutSeconds int
}

type SuggestionConfig struct {
	DefaultLimit               int
	MaxLimit                   int
	DefaultAlgorithm           entity.SuggestionAlgorithm
	HybridContentWeight        float64
	HybridCollaborativeWeight  float64
	MinImdbRating              float64
	SerendipityMinRating       float64
	ContentBasedMinWatches     int
	CollaborativeMinWatches    int
	ContentPreferenceThreshold float64
}

type SQSConfig struct {
	QueueURL    string
	WorkerCount int
}

type AWSConfig struct {
	Region   string
	Endpoint string
}

type PostgresConfig struct {
	DSN string
}

type AuthConfig struct {
	Pepper      string
	Secret      string
	ExpiryHours int
}

type OtelConfig struct {
	Endpoint    string
	ServiceName string
}

type ServerConfig struct {
	Port        int
	MetricsPort int
}

type LogConfig struct {
	Pretty bool
}

func Load() (*Config, error) {
	neo4jURI := getEnv("NEO4J_URI", "bolt://localhost:7687")
	neo4jUser := getEnv("NEO4J_USERNAME", "neo4j")
	neo4jPass := getEnv("NEO4J_PASSWORD", "password")
	neo4jDB := getEnv("NEO4J_DATABASE", "neo4j")

	omdbBase := getEnv("OMDB_BASE_URL", "http://www.omdbapi.com")
	omdbKey := getEnv("OMDB_API_KEY", "")
	omdbTimeout, _ := strconv.Atoi(getEnv("OMDB_TIMEOUT_SECONDS", "10"))

	defaultLimit, _ := strconv.Atoi(getEnv("SUGGESTION_DEFAULT_LIMIT", "10"))
	maxLimit, _ := strconv.Atoi(getEnv("SUGGESTION_MAX_LIMIT", "50"))
	hybridContent, _ := strconv.ParseFloat(getEnv("SUGGESTION_HYBRID_CONTENT_WEIGHT", "0.5"), 64)
	hybridCollab, _ := strconv.ParseFloat(getEnv("SUGGESTION_HYBRID_COLLABORATIVE_WEIGHT", "0.5"), 64)
	minRating, _ := strconv.ParseFloat(getEnv("SUGGESTION_MIN_IMDB_RATING", "6.0"), 64)
	serendipityRating, _ := strconv.ParseFloat(getEnv("SUGGESTION_SERENDIPITY_MIN_RATING", "5.0"), 64)
	contentBasedMin, _ := strconv.Atoi(getEnv("SUGGESTION_CONTENT_BASED_MIN_WATCHES", "5"))
	collaborativeMin, _ := strconv.Atoi(getEnv("SUGGESTION_COLLABORATIVE_MIN_WATCHES", "20"))
	contentPref, _ := strconv.ParseFloat(getEnv("SUGGESTION_CONTENT_PREFERENCE_THRESHOLD", "0.7"), 64)

	sqsQueueURL := getEnv("SQS_QUEUE_URL", "http://localhost:4566/000000000000/movie-import")
	sqsWorkers, _ := strconv.Atoi(getEnv("SQS_WORKER_COUNT", "5"))

	awsRegion := getEnv("AWS_REGION", "us-east-1")
	awsEndpoint := getEnv("AWS_ENDPOINT", "")

	postgresDSN := getEnv("POSTGRES_DSN", "postgres://postgres:password@localhost:5432/movie_suggestion?sslmode=disable")
	argon2Pepper := getEnv("ARGON2_PEPPER", "movie-suggestion-123456")
	jwtSecret := getEnv("JWT_SECRET", "dev-secret")
	jwtExpiryHours, _ := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", "24"))

	otelEndpoint := getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
	otelService := getEnv("OTEL_SERVICE_NAME", "movie-suggestion")

	serverPort, _ := strconv.Atoi(getEnv("SERVER_PORT", "8080"))
	metricsPort, _ := strconv.Atoi(getEnv("METRICS_PORT", "9090"))

	logPretty := getEnv("LOG_PRETTY", "false") == "true"

	return &Config{
		Neo4j: Neo4jConfig{
			URI:      neo4jURI,
			Username: neo4jUser,
			Password: neo4jPass,
			Database: neo4jDB,
		},
		OMDB: OMDBConfig{
			BaseURL:        omdbBase,
			APIKey:         omdbKey,
			TimeoutSeconds: omdbTimeout,
		},
		Suggestion: SuggestionConfig{
			DefaultLimit:               defaultLimit,
			MaxLimit:                   maxLimit,
			DefaultAlgorithm:           entity.AlgorithmPopular,
			HybridContentWeight:        hybridContent,
			HybridCollaborativeWeight:  hybridCollab,
			MinImdbRating:              minRating,
			SerendipityMinRating:       serendipityRating,
			ContentBasedMinWatches:     contentBasedMin,
			CollaborativeMinWatches:    collaborativeMin,
			ContentPreferenceThreshold: contentPref,
		},
		SQS: SQSConfig{
			QueueURL:    sqsQueueURL,
			WorkerCount: sqsWorkers,
		},
		AWS: AWSConfig{
			Region:   awsRegion,
			Endpoint: awsEndpoint,
		},
		Postgres: PostgresConfig{
			DSN: postgresDSN,
		},
		Auth: AuthConfig{
			Pepper:      argon2Pepper,
			Secret:      jwtSecret,
			ExpiryHours: jwtExpiryHours,
		},
		Otel: OtelConfig{
			Endpoint:    otelEndpoint,
			ServiceName: otelService,
		},
		Server: ServerConfig{
			Port:        serverPort,
			MetricsPort: metricsPort,
		},
		Log: LogConfig{
			Pretty: logPretty,
		},
	}, nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
