package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/WilliamCesarSantos/movie-suggestion/config"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/application/suggestion"
	appusecase "github.com/WilliamCesarSantos/movie-suggestion/internal/application/usecase"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/domain/entity"
)

type mockUserRepository struct {
	user *entity.User
	err  error
}

func (m *mockUserRepository) Create(ctx context.Context, user *entity.User) error { return nil }
func (m *mockUserRepository) FindByID(ctx context.Context, id string) (*entity.User, error) {
	return m.user, m.err
}
func (m *mockUserRepository) UpdateProfile(ctx context.Context, user *entity.User) error { return nil }
func (m *mockUserRepository) RecordWatched(ctx context.Context, userID, movieID string, userRating float64, reaction string) error {
	return nil
}

type mockSuggestionRepository struct {
	movies []*entity.Movie
	err    error
}

func (m *mockSuggestionRepository) FindPopular(ctx context.Context, userID string, limit int, minRating float64) ([]*entity.Movie, error) {
	return m.movies, m.err
}
func (m *mockSuggestionRepository) FindContentBased(ctx context.Context, userID string, limit int, minRating float64) ([]*entity.Movie, error) {
	return m.movies, m.err
}
func (m *mockSuggestionRepository) FindCollaborative(ctx context.Context, userID string, limit int, minRating float64) ([]*entity.Movie, error) {
	return m.movies, m.err
}
func (m *mockSuggestionRepository) FindHybrid(ctx context.Context, userID string, limit int, minRating float64, contentWeight, collaborativeWeight float64) ([]*entity.Movie, error) {
	return m.movies, m.err
}
func (m *mockSuggestionRepository) FindSerendipity(ctx context.Context, userID string, limit int, minRating float64) ([]*entity.Movie, error) {
	return m.movies, m.err
}

func TestSuggestMoviesUseCase_Execute(t *testing.T) {
	cfg := config.SuggestionConfig{
		DefaultLimit:            10,
		MaxLimit:                50,
		MinImdbRating:           6.0,
		SerendipityMinRating:    5.0,
		ContentBasedMinWatches:  5,
		CollaborativeMinWatches: 20,
	}

	selector := suggestion.NewAlgorithmSelector(0.7, 20, 5)

	t.Run("returns error when user not found", func(t *testing.T) {
		userRepo := &mockUserRepository{err: entity.ErrUserNotFound}
		suggRepo := &mockSuggestionRepository{}
		dispatcher := suggestion.NewAlgorithmDispatcher(suggRepo)
		uc := appusecase.NewSuggestMoviesUseCase(userRepo, suggRepo, selector, dispatcher, cfg)
		_, err := uc.Execute(context.Background(), "user1", 10, nil)
		if !errors.Is(err, entity.ErrUserNotFound) {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})

	t.Run("returns popular movies for new user", func(t *testing.T) {
		movies := []*entity.Movie{{ID: "m1", Title: "Movie 1"}}
		userRepo := &mockUserRepository{user: &entity.User{ID: "user1", WatchCount: 0}}
		suggRepo := &mockSuggestionRepository{movies: movies}
		dispatcher := suggestion.NewAlgorithmDispatcher(suggRepo)
		uc := appusecase.NewSuggestMoviesUseCase(userRepo, suggRepo, selector, dispatcher, cfg)
		result, err := uc.Execute(context.Background(), "user1", 10, nil)
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
		suggRepo := &mockSuggestionRepository{movies: movies}
		dispatcher := suggestion.NewAlgorithmDispatcher(suggRepo)
		uc := appusecase.NewSuggestMoviesUseCase(userRepo, suggRepo, selector, dispatcher, cfg)
		algo := entity.AlgorithmSerendipity
		result, err := uc.Execute(context.Background(), "user1", 10, &algo)
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
		suggRepo := &mockSuggestionRepository{movies: movies}
		dispatcher := suggestion.NewAlgorithmDispatcher(suggRepo)
		uc := appusecase.NewSuggestMoviesUseCase(userRepo, suggRepo, selector, dispatcher, cfg)
		result, err := uc.Execute(context.Background(), "user1", 0, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_ = result
	})
}
