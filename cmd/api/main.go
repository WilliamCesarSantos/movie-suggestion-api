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
	appusecase "github.com/WilliamCesarSantos/movie-suggestion/internal/application/usecase"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/application/suggestion"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/infrastructure/http/handler"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/infrastructure/http/middleware"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/infrastructure/http/router"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/infrastructure/lambda"
	neo4jinfra "github.com/WilliamCesarSantos/movie-suggestion/internal/infrastructure/neo4j"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/infrastructure/observability"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/infrastructure/omdb"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/infrastructure/sqs"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awslambda "github.com/aws/aws-sdk-go-v2/service/lambda"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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

	// OTEL tracer
	shutdownTracer, err := observability.InitTracer(cfg.Otel)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to initialize tracer, continuing without tracing")
	}

	// Prometheus metrics
	metrics := observability.NewMetrics()

	// Neo4j driver
	neo4jDriver, err := neo4j.NewDriverWithContext(cfg.Neo4j.URI, neo4j.BasicAuth(cfg.Neo4j.Username, cfg.Neo4j.Password, ""))
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create neo4j driver")
	}
	defer neo4jDriver.Close(context.Background())

	// Repositories
	movieRepo := neo4jinfra.NewMovieRepository(neo4jDriver, cfg.Neo4j.Database)
	userRepo := neo4jinfra.NewUserRepository(neo4jDriver, cfg.Neo4j.Database)
	suggestionRepo := neo4jinfra.NewSuggestionRepository(neo4jDriver, cfg.Neo4j.Database)

	// OMDB client
	omdbClient := omdb.NewClient(cfg.OMDB.BaseURL, cfg.OMDB.APIKey, cfg.OMDB.TimeoutSeconds)

	// AWS config
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.AWS.Region),
	)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to load AWS config")
	}

	// Lambda client with optional custom endpoint
	var lambdaClient *awslambda.Client
	if cfg.AWS.Endpoint != "" {
		lambdaClient = awslambda.NewFromConfig(awsCfg, func(o *awslambda.Options) {
			o.BaseEndpoint = aws.String(cfg.AWS.Endpoint)
		})
	} else {
		lambdaClient = awslambda.NewFromConfig(awsCfg)
	}

	// SQS client with optional custom endpoint
	var sqsClient *awssqs.Client
	if cfg.AWS.Endpoint != "" {
		sqsClient = awssqs.NewFromConfig(awsCfg, func(o *awssqs.Options) {
			o.BaseEndpoint = aws.String(cfg.AWS.Endpoint)
		})
	} else {
		sqsClient = awssqs.NewFromConfig(awsCfg)
	}

	authClient := lambda.NewAuthClient(lambdaClient, cfg.Lambda.AuthFunctionName)
	importClient := lambda.NewImportClient(lambdaClient, cfg.Lambda.ImportFunctionName)

	// Algorithm selector & dispatcher
	selector := suggestion.NewAlgorithmSelector(cfg.Suggestion.ContentPreferenceThreshold, cfg.Suggestion.CollaborativeMinWatches, cfg.Suggestion.ContentBasedMinWatches)
	dispatcher := suggestion.NewAlgorithmDispatcher(suggestionRepo)

	// Use cases
	suggestUC := appusecase.NewSuggestMoviesUseCase(userRepo, suggestionRepo, selector, dispatcher, cfg.Suggestion)
	importUC := appusecase.NewImportMoviesUseCase(importClient)
	manageUserUC := appusecase.NewManageUserUseCase(userRepo, selector)
	updateProfileUC := appusecase.NewUpdateUserProfileUseCase(userRepo)
	processImportUC := appusecase.NewProcessMovieImportUseCase(movieRepo, omdbClient, metrics)

	// HTTP handlers
	userHandler := handler.NewUserHandler(manageUserUC, suggestUC, updateProfileUC)
	movieHandler := handler.NewMovieHandler(movieRepo)
	adminHandler := handler.NewAdminHandler(importUC)
	healthHandler := handler.NewHealthHandler(neo4jDriver, cfg.Neo4j.Database)

	// Auth middleware
	authMiddleware := middleware.NewAuthMiddleware(authClient)

	// Router
	r := router.NewRouter(userHandler, movieHandler, adminHandler, healthHandler, authMiddleware, metrics)

	// SQS consumer
	consumer := sqs.NewConsumer(sqsClient, cfg.SQS.QueueURL, cfg.SQS.WorkerCount, processImportUC, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go consumer.Start(ctx)

	// Metrics server
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

	// HTTP server
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

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info().Msg("shutting down...")

	cancel() // stop SQS workers

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
