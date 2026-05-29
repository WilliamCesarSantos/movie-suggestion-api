package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/WilliamCesarSantos/movie-suggestion/app/internal/infrastructure/observability"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func ObservabilityMiddleware(metrics *observability.Metrics) func(http.Handler) http.Handler {
	tracer := otel.Tracer("movie-suggestion")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			correlationID := r.Header.Get("X-Correlation-ID")
			if correlationID == "" {
				correlationID = uuid.New().String()
			}
			w.Header().Set("X-Correlation-ID", correlationID)

			ctx, span := tracer.Start(r.Context(), r.Method+" "+r.URL.Path)
			span.SetAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.path", r.URL.Path),
				attribute.String("correlation.id", correlationID),
			)
			defer span.End()

			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			start := time.Now()
			next.ServeHTTP(rw, r.WithContext(ctx))
			duration := time.Since(start).Seconds()

			statusStr := strconv.Itoa(rw.status)
			metrics.HttpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, statusStr).Inc()
			metrics.HttpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
		})
	}
}
