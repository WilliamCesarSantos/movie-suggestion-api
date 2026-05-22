package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/WilliamCesarSantos/movie-suggestion/config"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/application/suggestion"
	appusecase "github.com/WilliamCesarSantos/movie-suggestion/internal/application/usecase"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/infrastructure/auth"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/infrastructure/http/handler"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/infrastructure/http/middleware"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/infrastructure/http/router"
	neo4jinfra "github.com/WilliamCesarSantos/movie-suggestion/internal/infrastructure/neo4j"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/infrastructure/observability"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/infrastructure/omdb"
	postgresinfra "github.com/WilliamCesarSantos/movie-suggestion/internal/infrastructure/postgres"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/infrastructure/sqs"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	if cfg.Log.Pretty {
		logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	shutdownTracer, err := observability.InitTracer(cfg.Otel)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to initialize tracer, continuing without tracing")
	}

	metrics := observability.NewMetrics()

	neo4jDriver, err := neo4j.NewDriverWithContext(cfg.Neo4j.URI, neo4j.BasicAuth(cfg.Neo4j.Username, cfg.Neo4j.Password, ""))
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create neo4j driver")
	}
	defer neo4jDriver.Close(context.Background())

	db, err := gorm.Open(postgres.Open(cfg.Postgres.DSN), &gorm.Config{})
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to postgres")
	}

	movieRepo := neo4jinfra.NewMovieRepository(neo4jDriver, cfg.Neo4j.Database)
	userRepo := neo4jinfra.NewUserRepository(neo4jDriver, cfg.Neo4j.Database)
	suggestionRepo := neo4jinfra.NewSuggestionRepository(neo4jDriver, cfg.Neo4j.Database)
	authUserRepo := postgresinfra.NewAuthUserRepository(db)

	omdbClient := omdb.NewClient(cfg.OMDB.BaseURL, cfg.OMDB.APIKey, cfg.OMDB.TimeoutSeconds)
	omdbSearcher := omdb.NewSearcherAdapter(omdbClient)

	passwordService := auth.NewPasswordService(cfg.Auth.Pepper)
	jwtService := auth.NewJWTService(cfg.Auth.Secret, cfg.Auth.ExpiryHours)

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), awsconfig.WithRegion(cfg.AWS.Region))
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to load AWS config")
	}

	var sqsClient *awssqs.Client
	if cfg.AWS.Endpoint != "" {
		sqsClient = awssqs.NewFromConfig(awsCfg, func(o *awssqs.Options) {
			o.BaseEndpoint = aws.String(cfg.AWS.Endpoint)
		})
	} else {
		sqsClient = awssqs.NewFromConfig(awsCfg)
	}
	publisher := sqs.NewPublisher(sqsClient, cfg.SQS.QueueURL)

	selector := suggestion.NewAlgorithmSelector(cfg.Suggestion.ContentPreferenceThreshold, cfg.Suggestion.CollaborativeMinWatches, cfg.Suggestion.ContentBasedMinWatches)
	dispatcher := suggestion.NewAlgorithmDispatcher(suggestionRepo)

	suggestUC := appusecase.NewSuggestMoviesUseCase(userRepo, suggestionRepo, selector, dispatcher, cfg.Suggestion)
	importUC := appusecase.NewImportMoviesUseCase(omdbSearcher, publisher)
	manageUserUC := appusecase.NewManageUserUseCase(userRepo, selector)
	updateProfileUC := appusecase.NewUpdateUserProfileUseCase(userRepo)
	processImportUC := appusecase.NewProcessMovieImportUseCase(movieRepo, omdbClient, metrics)
	getMovieUC := appusecase.NewGetMovieUseCase(movieRepo)
	loginUC := appusecase.NewLoginUseCase(authUserRepo, passwordService, jwtService)

	userHandler := handler.NewUserHandler(manageUserUC, suggestUC, updateProfileUC, authUserRepo, passwordService)
	movieHandler := handler.NewMovieHandler(getMovieUC, manageUserUC)
	importHandler := handler.NewImportHandler(importUC)
	authHandler := handler.NewAuthHandler(loginUC)
	healthHandler := handler.NewHealthHandler(neo4jDriver, cfg.Neo4j.Database)

	authMiddleware := middleware.NewAuthMiddleware(jwtService)

	r := router.NewRouter(userHandler, movieHandler, importHandler, authHandler, healthHandler, authMiddleware, metrics)

	consumer := sqs.NewConsumer(sqsClient, cfg.SQS.QueueURL, cfg.SQS.WorkerCount, processImportUC, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go consumer.Start(ctx)

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.MetricsPort),
		Handler: metricsMux,
	}
	go func() {
		logger.Info().Int("port", cfg.Server.MetricsPort).Msg("metrics server starting")
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("metrics server error")
		}
	}()

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: r,
	}

	go func() {
		logger.Info().Int("port", cfg.Server.Port).Msg("API server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("API server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info().Msg("shutting down...")

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("API server shutdown error")
	}
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("metrics server shutdown error")
	}
	if shutdownTracer != nil {
		if err := shutdownTracer(shutdownCtx); err != nil {
			logger.Error().Err(err).Msg("tracer shutdown error")
		}
	}
	logger.Info().Msg("shutdown complete")
}
