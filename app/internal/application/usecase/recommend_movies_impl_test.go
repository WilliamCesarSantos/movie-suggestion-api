package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/config"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/application/recommendation"
	appusecase "github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/application/usecase"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
)

type mockUserRepository struct {
	user *entity.User
	err  error
}

func (m *mockUserRepository) Create(ctx context.Context, user *entity.User) error { return nil }
func (m *mockUserRepository) FindByID(ctx context.Context, id string) (*entity.User, error) {
	return m.user, m.err
}
func (m *mockUserRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	return m.user, m.err
}
func (m *mockUserRepository) UpdateProfile(ctx context.Context, user *entity.User) error { return nil }
func (m *mockUserRepository) RecordWatched(ctx context.Context, userID, movieID string, userRating float64, reaction string) error {
	return nil
}

type mockRecommendationRepository struct {
	movies         []*entity.Movie
	err            error
	receivedTitles []string
}

func (m *mockRecommendationRepository) FindPopular(ctx context.Context, userID string, limit int, minRating float64, title string) ([]*entity.Movie, error) {
	m.receivedTitles = append(m.receivedTitles, title)
	return m.movies, m.err
}
func (m *mockRecommendationRepository) FindContentBased(ctx context.Context, userID string, limit int, minRating float64, title string) ([]*entity.Movie, error) {
	m.receivedTitles = append(m.receivedTitles, title)
	return m.movies, m.err
}
func (m *mockRecommendationRepository) FindCollaborative(ctx context.Context, userID string, limit int, minRating float64, title string) ([]*entity.Movie, error) {
	m.receivedTitles = append(m.receivedTitles, title)
	return m.movies, m.err
}
func (m *mockRecommendationRepository) FindHybrid(ctx context.Context, userID string, limit int, minRating float64, contentWeight, collaborativeWeight float64, title string) ([]*entity.Movie, error) {
	m.receivedTitles = append(m.receivedTitles, title)
	return m.movies, m.err
}
func (m *mockRecommendationRepository) FindSerendipity(ctx context.Context, userID string, limit int, minRating float64, title string) ([]*entity.Movie, error) {
	m.receivedTitles = append(m.receivedTitles, title)
	return m.movies, m.err
}

type fallbackRecommendationRepository struct {
	popularMovies       []*entity.Movie
	collaborativeMovies []*entity.Movie
}

func (m *fallbackRecommendationRepository) FindPopular(ctx context.Context, userID string, limit int, minRating float64, title string) ([]*entity.Movie, error) {
	return m.popularMovies, nil
}

func (m *fallbackRecommendationRepository) FindContentBased(ctx context.Context, userID string, limit int, minRating float64, title string) ([]*entity.Movie, error) {
	return nil, nil
}

func (m *fallbackRecommendationRepository) FindCollaborative(ctx context.Context, userID string, limit int, minRating float64, title string) ([]*entity.Movie, error) {
	return m.collaborativeMovies, nil
}

func (m *fallbackRecommendationRepository) FindHybrid(ctx context.Context, userID string, limit int, minRating float64, contentWeight, collaborativeWeight float64, title string) ([]*entity.Movie, error) {
	return nil, nil
}

func (m *fallbackRecommendationRepository) FindSerendipity(ctx context.Context, userID string, limit int, minRating float64, title string) ([]*entity.Movie, error) {
	return nil, nil
}

func TestRecommendMoviesUseCase_Execute(t *testing.T) {
	cfg := config.RecommendationConfig{
		DefaultLimit:            10,
		MaxLimit:                50,
		MinImdbRating:           6.0,
		SerendipityMinRating:    5.0,
		ContentBasedMinWatches:  5,
		CollaborativeMinWatches: 20,
	}

	selector := recommendation.NewAlgorithmSelector(0.7, 20, 5)

	t.Run("returns error when user not found", func(t *testing.T) {
		userRepo := &mockUserRepository{err: entity.ErrUserNotFound}
		suggRepo := &mockRecommendationRepository{}
		dispatcher := recommendation.NewAlgorithmDispatcher(suggRepo)
		uc := appusecase.NewRecommendMoviesUseCase(userRepo, suggRepo, selector, dispatcher, cfg)
		_, err := uc.Execute(context.Background(), "user@example.com", 10, nil, "")
		if !errors.Is(err, entity.ErrUserNotFound) {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})

	t.Run("returns popular movies for new user", func(t *testing.T) {
		movies := []*entity.Movie{{ID: "m1", Title: "Movie 1"}}
		userRepo := &mockUserRepository{user: &entity.User{ID: "user1", WatchCount: 0}}
		suggRepo := &mockRecommendationRepository{movies: movies}
		dispatcher := recommendation.NewAlgorithmDispatcher(suggRepo)
		uc := appusecase.NewRecommendMoviesUseCase(userRepo, suggRepo, selector, dispatcher, cfg)
		result, err := uc.Execute(context.Background(), "user@example.com", 10, nil, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("expected 1 movie, got %d", len(result))
		}
	})

	t.Run("respects algorithm override", func(t *testing.T) {
		movies := []*entity.Movie{{ID: "m1", Title: "Movie 1"}}
		userRepo := &mockUserRepository{user: &entity.User{ID: "user1", WatchCount: 0}}
		suggRepo := &mockRecommendationRepository{movies: movies}
		dispatcher := recommendation.NewAlgorithmDispatcher(suggRepo)
		uc := appusecase.NewRecommendMoviesUseCase(userRepo, suggRepo, selector, dispatcher, cfg)
		algo := entity.AlgorithmSerendipity
		result, err := uc.Execute(context.Background(), "user@example.com", 10, &algo, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("expected 1 movie, got %d", len(result))
		}
	})

	t.Run("uses default limit when limit is 0", func(t *testing.T) {
		movies := []*entity.Movie{{ID: "m1", Title: "Movie 1"}}
		userRepo := &mockUserRepository{user: &entity.User{ID: "user1", WatchCount: 0}}
		suggRepo := &mockRecommendationRepository{movies: movies}
		dispatcher := recommendation.NewAlgorithmDispatcher(suggRepo)
		uc := appusecase.NewRecommendMoviesUseCase(userRepo, suggRepo, selector, dispatcher, cfg)
		result, err := uc.Execute(context.Background(), "user@example.com", 0, nil, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_ = result
	})

	t.Run("falls back to popular when selected algorithm returns empty", func(t *testing.T) {
		popularMovies := []*entity.Movie{{ID: "m-pop-1", Title: "Popular Movie"}}
		userRepo := &mockUserRepository{user: &entity.User{ID: "user1", WatchCount: 20}}
		suggRepo := &fallbackRecommendationRepository{
			popularMovies:       popularMovies,
			collaborativeMovies: []*entity.Movie{},
		}
		dispatcher := recommendation.NewAlgorithmDispatcher(suggRepo)
		uc := appusecase.NewRecommendMoviesUseCase(userRepo, suggRepo, selector, dispatcher, cfg)

		result, err := uc.Execute(context.Background(), "user@example.com", 10, nil, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 || result[0].ID != "m-pop-1" {
			t.Fatalf("expected fallback popular movie, got %#v", result)
		}
	})

	t.Run("passes title filter to repository", func(t *testing.T) {
		movies := []*entity.Movie{{ID: "m1", Title: "Movie 1"}}
		userRepo := &mockUserRepository{user: &entity.User{ID: "user1", WatchCount: 0}}
		suggRepo := &mockRecommendationRepository{movies: movies}
		dispatcher := recommendation.NewAlgorithmDispatcher(suggRepo)
		uc := appusecase.NewRecommendMoviesUseCase(userRepo, suggRepo, selector, dispatcher, cfg)

		_, err := uc.Execute(context.Background(), "user@example.com", 10, nil, "matrix")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(suggRepo.receivedTitles) == 0 || suggRepo.receivedTitles[0] != "matrix" {
			t.Fatalf("expected title filter 'matrix', got %#v", suggRepo.receivedTitles)
		}
	})
}
