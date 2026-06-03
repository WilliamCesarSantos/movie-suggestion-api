package suggestion

import "github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"

type AlgorithmSelector struct {
	contentPreferenceThreshold float64
	collaborativeMinWatches    int
	contentBasedMinWatches     int
}

func NewAlgorithmSelector(contentPreferenceThreshold float64, collaborativeMinWatches, contentBasedMinWatches int) *AlgorithmSelector {
	return &AlgorithmSelector{
		contentPreferenceThreshold: contentPreferenceThreshold,
		collaborativeMinWatches:    collaborativeMinWatches,
		contentBasedMinWatches:     contentBasedMinWatches,
	}
}

func (s *AlgorithmSelector) Select(user entity.User) entity.SuggestionAlgorithm {
	if user.WatchCount < s.contentBasedMinWatches {
		return entity.AlgorithmPopular
	}
	if user.WatchCount < s.collaborativeMinWatches {
		return entity.AlgorithmContentBased
	}
	return entity.AlgorithmCollaborative
}
