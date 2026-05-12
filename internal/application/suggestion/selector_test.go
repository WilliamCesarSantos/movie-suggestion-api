package suggestion_test

import (
	"testing"

	"github.com/WilliamCesarSantos/movie-suggestion/internal/application/suggestion"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/domain/entity"
)

func TestAlgorithmSelector_Rules(t *testing.T) {
	selector := suggestion.NewAlgorithmSelector(0.7, 20, 5)

	tests := []struct {
		name     string
		user     entity.User
		expected entity.SuggestionAlgorithm
	}{
		{
			name:     "Rule 1: watchCount < 5 → POPULAR",
			user:     entity.User{WatchCount: 0},
			expected: entity.AlgorithmPopular,
		},
		{
			name:     "Rule 1: watchCount = 4 → POPULAR",
			user:     entity.User{WatchCount: 4},
			expected: entity.AlgorithmPopular,
		},
		{
			name:     "Rule 2: watchCount = 5 → CONTENT_BASED",
			user:     entity.User{WatchCount: 5},
			expected: entity.AlgorithmContentBased,
		},
		{
			name:     "Rule 2: watchCount = 19 → CONTENT_BASED",
			user:     entity.User{WatchCount: 19},
			expected: entity.AlgorithmContentBased,
		},
		{
			name:     "Rule 3: watchCount = 20 → COLLABORATIVE",
			user:     entity.User{WatchCount: 20},
			expected: entity.AlgorithmCollaborative,
		},
		{
			name:     "Rule 3: watchCount = 100 → COLLABORATIVE",
			user:     entity.User{WatchCount: 100},
			expected: entity.AlgorithmCollaborative,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selector.Select(tt.user)
			if got != tt.expected {
				t.Errorf("Select() = %v, want %v", got, tt.expected)
			}
		})
	}
}
