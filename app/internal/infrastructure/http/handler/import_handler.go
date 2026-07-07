package handler

import (
	"encoding/json"
	"net/http"

	domainusecase "github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/usecase"
	"github.com/rs/zerolog/log"
)

type ImportHandler struct {
	importUC domainusecase.ImportMoviesUseCase
}

func NewImportHandler(importUC domainusecase.ImportMoviesUseCase) *ImportHandler {
	return &ImportHandler{importUC: importUC}
}

type triggerImportRequest struct {
	SearchTerms []string `json:"searchTerms"`
	MaxPages    int      `json:"maxPages"`
}

func (h *ImportHandler) TriggerImport(w http.ResponseWriter, r *http.Request) {
	logger := log.Ctx(r.Context()).With().Str("logger", "http.import").Logger()
	logger.Info().Msg("import request received")

	var req triggerImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn().Msg("import request rejected: invalid body")
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.SearchTerms) == 0 {
		logger.Warn().Msg("import request rejected: searchTerms required")
		http.Error(w, "searchTerms required", http.StatusBadRequest)
		return
	}
	if req.MaxPages <= 0 {
		req.MaxPages = 1
	}
	logger.Info().Int("searchTermsCount", len(req.SearchTerms)).Int("maxPages", req.MaxPages).Msg("triggering movie import")
	if err := h.importUC.Execute(r.Context(), req.SearchTerms, req.MaxPages); err != nil {
		logger.Error().Err(err).Msg("failed to trigger movie import")
		http.Error(w, "failed to trigger import", http.StatusInternalServerError)
		return
	}
	logger.Info().Msg("movie import triggered")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "import triggered"})
}
