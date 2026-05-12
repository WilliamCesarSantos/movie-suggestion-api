package entity

import "time"

type SuggestionAlgorithm string

const (
	AlgorithmPopular       SuggestionAlgorithm = "POPULAR"
	AlgorithmContentBased  SuggestionAlgorithm = "CONTENT_BASED"
	AlgorithmCollaborative SuggestionAlgorithm = "COLLABORATIVE"
	AlgorithmHybrid        SuggestionAlgorithm = "HYBRID"
	AlgorithmSerendipity   SuggestionAlgorithm = "SERENDIPITY"
)

type User struct {
	ID               string
	Name             string
	Email            string
	CreatedAt        time.Time
	CurrentAlgorithm SuggestionAlgorithm
	WatchCount       int
	LikeCount        int
	DislikeCount     int
}
