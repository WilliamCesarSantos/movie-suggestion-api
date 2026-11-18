package middleware

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/auth"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/observability"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestRequestLogsCarryRequiredFields(t *testing.T) {
	var buf bytes.Buffer
	previous := log.Logger
	log.Logger = zerolog.New(&buf).With().Timestamp().Str("pod", "test-pod").Logger()
	t.Cleanup(func() { log.Logger = previous })

	jwtService := auth.NewJWTService("test-secret", 1)
	token, _, err := jwtService.Generate("user-123", "alice@example.com", []string{"movies:read"})
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	r := chi.NewRouter()
	r.Use(ObservabilityMiddleware(observability.NewMetrics()))
	r.With(NewAuthMiddleware(jwtService).Authenticate).Get("/api/v1/movies/{id}", func(w http.ResponseWriter, r *http.Request) {
		log.Ctx(r.Context()).Info().Str("logger", "http.test").Msg("handler log")
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/movies/42", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Correlation-ID", "cid-123")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	entries := map[string]map[string]any{}
	scanner := bufio.NewScanner(&buf)
	for scanner.Scan() {
		line := scanner.Text()
		for _, key := range []string{"userEmail", "correlationId", "pod", "logger"} {
			if n := strings.Count(line, `"`+key+`":`); n > 1 {
				t.Errorf("duplicated key %q in log line: %s", key, line)
			}
		}
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("log line is not JSON: %s", line)
		}
		entries[entry["message"].(string)] = entry
	}

	required := map[string]any{
		"correlationId": "cid-123",
		"pod":           "test-pod",
		"userEmail":     "alice@example.com",
	}
	for _, msg := range []string{"handler log", "request completed"} {
		entry, ok := entries[msg]
		if !ok {
			t.Fatalf("missing log %q", msg)
		}
		for key, want := range required {
			if entry[key] != want {
				t.Errorf("%q: expected %s=%v, got %v", msg, key, want, entry[key])
			}
		}
		for _, key := range []string{"time", "level", "logger"} {
			if entry[key] == nil || entry[key] == "" {
				t.Errorf("%q: missing %s", msg, key)
			}
		}
	}

	access := entries["request completed"]
	if access["route"] != "/api/v1/movies/{id}" {
		t.Errorf("expected route pattern, got %v", access["route"])
	}
	if access["logger"] != "http.access" {
		t.Errorf("expected logger http.access, got %v", access["logger"])
	}
}
