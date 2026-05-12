package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type HealthHandler struct {
	neo4jDriver neo4j.DriverWithContext
	database    string
}

func NewHealthHandler(neo4jDriver neo4j.DriverWithContext, database string) *HealthHandler {
	return &HealthHandler{neo4jDriver: neo4jDriver, database: database}
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	status := "ok"
	neo4jStatus := "ok"

	if err := h.neo4jDriver.VerifyConnectivity(ctx); err != nil {
		neo4jStatus = "unavailable"
		status = "degraded"
	}

	resp := map[string]any{
		"status":    status,
		"timestamp": time.Now().Format(time.RFC3339),
		"services": map[string]string{
			"neo4j": neo4jStatus,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if status != "ok" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	json.NewEncoder(w).Encode(resp)
}
