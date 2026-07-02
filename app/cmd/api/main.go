package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/config"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/application/suggestion"
	appusecase "github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/application/usecase"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/repository"
	domainusecase "github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/usecase"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/auth"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/http/handler"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/http/middleware"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/http/router"
	neo4jinfra "github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/neo4j"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/observability"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/omdb"
	postgresinfra "github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/postgres"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/sqs"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Config

func provideConfig() (*config.Config, error) {
	return config.Load()
}

func provideLogger(cfg *config.Config) zerolog.Logger {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	if cfg.Log.Pretty {
		logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()
	}
	log.Logger = logger
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	return logger
}

// Infrastructure

func provideNeo4jDriver(lc fx.Lifecycle, cfg *config.Config) (neo4j.DriverWithContext, error) {
	driver, err := neo4j.NewDriverWithContext(cfg.Neo4j.URI, neo4j.BasicAuth(cfg.Neo4j.Username, cfg.Neo4j.Password, ""))
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return driver.Close(ctx)
		},
	})
	return driver, nil
}

func providePostgresDB(cfg *config.Config) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(cfg.Postgres.DSN), &gorm.Config{})
}

func provideAWSConfig(cfg *config.Config) (aws.Config, error) {
	return awsconfig.LoadDefaultConfig(context.Background(), awsconfig.WithRegion(cfg.AWS.Region))
}

func provideSQSClient(cfg *config.Config, awsCfg aws.Config) *awssqs.Client {
	if cfg.AWS.Endpoint != "" {
		return awssqs.NewFromConfig(awsCfg, func(o *awssqs.Options) {
			o.BaseEndpoint = aws.String(cfg.AWS.Endpoint)
		})
	}
	return awssqs.NewFromConfig(awsCfg)
}

func provideMetrics() *observability.Metrics {
	return observability.NewMetrics()
}

func provideMovieRepo(driver neo4j.DriverWithContext, cfg *config.Config) repository.MovieRepository {
	return neo4jinfra.NewMovieRepository(driver, cfg.Neo4j.Database)
}

func provideUserRepo(driver neo4j.DriverWithContext, cfg *config.Config) repository.UserRepository {
	return neo4jinfra.NewUserRepository(driver, cfg.Neo4j.Database)
}

func provideSuggestionRepo(driver neo4j.DriverWithContext, cfg *config.Config) repository.SuggestionRepository {
	return neo4jinfra.NewSuggestionRepository(driver, cfg.Neo4j.Database)
}

func provideAuthUserRepo(db *gorm.DB) repository.AuthUserRepository {
	return postgresinfra.NewAuthUserRepository(db)
}

func provideOMDBClient(cfg *config.Config) *omdb.Client {
	return omdb.NewClient(cfg.OMDB.BaseURL, cfg.OMDB.APIKey, cfg.OMDB.TimeoutSeconds)
}

func provideOMDBSearcher(client *omdb.Client) domainusecase.OmdbSearcher {
	return omdb.NewSearcherAdapter(client)
}

func providePasswordService(cfg *config.Config) *auth.PasswordService {
	return auth.NewPasswordService(cfg.Auth.Pepper)
}

func provideJWTService(cfg *config.Config) *auth.JWTService {
	return auth.NewJWTService(cfg.Auth.Secret, cfg.Auth.ExpiryHours)
}

func provideSQSPublisher(client *awssqs.Client, cfg *config.Config) domainusecase.MovieImportPublisher {
	return sqs.NewPublisher(client, cfg.SQS.QueueURL)
}

// Application

func provideAlgorithmSelector(cfg *config.Config) *suggestion.AlgorithmSelector {
	return suggestion.NewAlgorithmSelector(
		cfg.Suggestion.ContentPreferenceThreshold,
		cfg.Suggestion.CollaborativeMinWatches,
		cfg.Suggestion.ContentBasedMinWatches,
	)
}

func provideAlgorithmDispatcher(repo repository.SuggestionRepository) *suggestion.AlgorithmDispatcher {
	return suggestion.NewAlgorithmDispatcher(repo)
}

func provideSuggestUseCase(
	userRepo repository.UserRepository,
	suggestionRepo repository.SuggestionRepository,
	selector *suggestion.AlgorithmSelector,
	dispatcher *suggestion.AlgorithmDispatcher,
	cfg *config.Config,
) domainusecase.SuggestMoviesUseCase {
	return appusecase.NewSuggestMoviesUseCase(userRepo, suggestionRepo, selector, dispatcher, cfg.Suggestion)
}

func provideImportUseCase(searcher domainusecase.OmdbSearcher, publisher domainusecase.MovieImportPublisher) domainusecase.ImportMoviesUseCase {
	return appusecase.NewImportMoviesUseCase(searcher, publisher)
}

func provideManageUserUseCase(repo repository.UserRepository, selector *suggestion.AlgorithmSelector) domainusecase.ManageUserUseCase {
	return appusecase.NewManageUserUseCase(repo, selector)
}

func provideUpdateProfileUseCase(repo repository.UserRepository) domainusecase.UpdateUserProfileUseCase {
	return appusecase.NewUpdateUserProfileUseCase(repo)
}

func provideProcessImportUseCase(repo repository.MovieRepository, client *omdb.Client, metrics *observability.Metrics) appusecase.ProcessMovieImportUseCase {
	return appusecase.NewProcessMovieImportUseCase(repo, client, metrics)
}

func provideGetMovieUseCase(repo repository.MovieRepository) domainusecase.GetMovieUseCase {
	return appusecase.NewGetMovieUseCase(repo)
}

func provideListUsersUseCase(repo repository.AuthUserRepository) domainusecase.ListUsersUseCase {
	return appusecase.NewListUsersUseCase(repo)
}

func provideLoginUseCase(repo repository.AuthUserRepository, ps *auth.PasswordService, js *auth.JWTService) domainusecase.LoginUseCase {
	return appusecase.NewLoginUseCase(repo, ps, js)
}

// HTTP

func provideUserHandler(
	manageUC domainusecase.ManageUserUseCase,
	suggestUC domainusecase.SuggestMoviesUseCase,
	updateUC domainusecase.UpdateUserProfileUseCase,
	listUsersUC domainusecase.ListUsersUseCase,
	authRepo repository.AuthUserRepository,
	ps *auth.PasswordService,
) *handler.UserHandler {
	return handler.NewUserHandler(manageUC, suggestUC, updateUC, listUsersUC, authRepo, ps)
}

func provideMovieHandler(getUC domainusecase.GetMovieUseCase, manageUC domainusecase.ManageUserUseCase) *handler.MovieHandler {
	return handler.NewMovieHandler(getUC, manageUC)
}

func provideImportHandler(importUC domainusecase.ImportMoviesUseCase) *handler.ImportHandler {
	return handler.NewImportHandler(importUC)
}

func provideAuthHandler(loginUC domainusecase.LoginUseCase) *handler.AuthHandler {
	return handler.NewAuthHandler(loginUC)
}

func provideHealthHandler(driver neo4j.DriverWithContext, cfg *config.Config) *handler.HealthHandler {
	return handler.NewHealthHandler(driver, cfg.Neo4j.Database)
}

func provideAuthMiddleware(js *auth.JWTService) *middleware.AuthMiddleware {
	return middleware.NewAuthMiddleware(js)
}

func provideRouter(
	userH *handler.UserHandler,
	movieH *handler.MovieHandler,
	importH *handler.ImportHandler,
	authH *handler.AuthHandler,
	healthH *handler.HealthHandler,
	authM *middleware.AuthMiddleware,
	metrics *observability.Metrics,
) http.Handler {
	return router.NewRouter(userH, movieH, importH, authH, healthH, authM, metrics)
}

func provideSQSConsumer(
	client *awssqs.Client,
	cfg *config.Config,
	processUC appusecase.ProcessMovieImportUseCase,
	logger zerolog.Logger,
) *sqs.Consumer {
	return sqs.NewConsumer(client, cfg.SQS.QueueURL, cfg.SQS.WorkerCount, processUC, logger)
}

// Lifecycle

type serverParams struct {
	fx.In
	LC       fx.Lifecycle
	Config   *config.Config
	Router   http.Handler
	Consumer *sqs.Consumer
	Logger   zerolog.Logger
}

func registerLifecycle(p serverParams) {
	var (
		apiServer      *http.Server
		metricsServer  *http.Server
		consumerCancel context.CancelFunc
	)

	p.LC.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			metricsMux := http.NewServeMux()
			metricsMux.Handle("/metrics", promhttp.Handler())
			metricsServer = &http.Server{
				Addr:    fmt.Sprintf(":%d", p.Config.Server.MetricsPort),
				Handler: metricsMux,
			}
			go func() {
				p.Logger.Info().Str("correlationId", "system").Int("port", p.Config.Server.MetricsPort).Msg("metrics server starting")
				if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					p.Logger.Error().Str("correlationId", "system").Err(err).Msg("metrics server error")
				}
			}()

			apiServer = &http.Server{
				Addr:    fmt.Sprintf(":%d", p.Config.Server.Port),
				Handler: p.Router,
			}
			go func() {
				p.Logger.Info().Str("correlationId", "system").Int("port", p.Config.Server.Port).Msg("API server starting")
				if err := apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					p.Logger.Error().Str("correlationId", "system").Err(err).Msg("API server error")
				}
			}()

			var consumerCtx context.Context
			consumerCtx, consumerCancel = context.WithCancel(context.Background())
			go p.Consumer.Start(consumerCtx)

			return nil
		},
		OnStop: func(ctx context.Context) error {
			p.Logger.Info().Str("correlationId", "system").Msg("shutting down...")
			consumerCancel()
			if err := apiServer.Shutdown(ctx); err != nil {
				p.Logger.Error().Str("correlationId", "system").Err(err).Msg("API server shutdown error")
			}
			if err := metricsServer.Shutdown(ctx); err != nil {
				p.Logger.Error().Str("correlationId", "system").Err(err).Msg("metrics server shutdown error")
			}
			return nil
		},
	})
}

func registerTracer(lc fx.Lifecycle, cfg *config.Config, logger zerolog.Logger) {
	var tracerShutdown func(context.Context) error
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			shutdown, err := observability.InitTracer(cfg.Otel)
			if err != nil {
				logger.Warn().Str("correlationId", "system").Err(err).Msg("failed to initialize tracer, continuing without tracing")
				return nil
			}
			tracerShutdown = shutdown
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if tracerShutdown != nil {
				return tracerShutdown(ctx)
			}
			return nil
		},
	})
}

// Modules

var configModule = fx.Module("config",
	fx.Provide(
		provideConfig,
		provideLogger,
	),
)

var infrastructureModule = fx.Module("infrastructure",
	fx.Provide(
		provideNeo4jDriver,
		providePostgresDB,
		provideAWSConfig,
		provideSQSClient,
		provideMetrics,
		provideMovieRepo,
		provideUserRepo,
		provideSuggestionRepo,
		provideAuthUserRepo,
		provideOMDBClient,
		provideOMDBSearcher,
		providePasswordService,
		provideJWTService,
		provideSQSPublisher,
	),
)

var applicationModule = fx.Module("application",
	fx.Provide(
		provideAlgorithmSelector,
		provideAlgorithmDispatcher,
		provideSuggestUseCase,
		provideImportUseCase,
		provideManageUserUseCase,
		provideUpdateProfileUseCase,
		provideProcessImportUseCase,
		provideGetMovieUseCase,
		provideLoginUseCase,
		provideListUsersUseCase,
	),
)

var httpModule = fx.Module("http",
	fx.Provide(
		provideUserHandler,
		provideMovieHandler,
		provideImportHandler,
		provideAuthHandler,
		provideHealthHandler,
		provideAuthMiddleware,
		provideRouter,
		provideSQSConsumer,
	),
)

func main() {
	fx.New(
		configModule,
		infrastructureModule,
		applicationModule,
		httpModule,
		fx.Invoke(registerTracer, registerLifecycle),
		fx.WithLogger(func() fxevent.Logger { return fxevent.NopLogger }),
	).Run()
}
