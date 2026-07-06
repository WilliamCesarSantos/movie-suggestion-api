package entity

import "time"

type RecommendationAlgorithm string

const (
	AlgorithmPopular       RecommendationAlgorithm = "POPULAR"
	AlgorithmContentBased  RecommendationAlgorithm = "CONTENT_BASED"
	AlgorithmCollaborative RecommendationAlgorithm = "COLLABORATIVE"
	AlgorithmHybrid        RecommendationAlgorithm = "HYBRID"
	AlgorithmSerendipity   RecommendationAlgorithm = "SERENDIPITY"
)

type User struct {
	ID               string
	Name             string
	Email            string
	CreatedAt        time.Time
	CurrentAlgorithm RecommendationAlgorithm
	WatchCount       int
	LikeCount        int
	DislikeCount     int
}
