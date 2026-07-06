package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	HttpRequestsTotal    *prometheus.CounterVec
	HttpRequestDuration  *prometheus.HistogramVec
	RecommendationsTotal *prometheus.CounterVec
	MovieImportTotal     *prometheus.CounterVec
	SqsMessagesProcessed *prometheus.CounterVec
	OmdbRequestDuration  *prometheus.HistogramVec
	Neo4jQueryDuration   *prometheus.HistogramVec
}

func NewMetrics() *Metrics {
	return &Metrics{
		HttpRequestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests",
		}, []string{"method", "path", "status"}),

		HttpRequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "path"}),

		RecommendationsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "recommendations_total",
			Help: "Total recommendations served",
		}, []string{"algorithm", "userId"}),

		MovieImportTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "movie_import_total",
			Help: "Total movie import events",
		}, []string{"status"}),

		SqsMessagesProcessed: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "sqs_messages_processed_total",
			Help: "Total SQS messages processed",
		}, []string{"status"}),

		OmdbRequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "omdb_request_duration_seconds",
			Help:    "OMDB request duration in seconds",
			Buckets: prometheus.DefBuckets,
		}, []string{"status"}),

		Neo4jQueryDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "neo4j_query_duration_seconds",
			Help:    "Neo4j query duration in seconds",
			Buckets: prometheus.DefBuckets,
		}, []string{"operation"}),
	}
}
