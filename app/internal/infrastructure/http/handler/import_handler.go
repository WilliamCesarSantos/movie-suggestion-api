package handler

import (
	"encoding/json"
	"net/http"

	domainusecase "github.com/WilliamCesarSantos/movie-suggestion/app/internal/domain/usecase"
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
	var req triggerImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.SearchTerms) == 0 {
		http.Error(w, "searchTerms required", http.StatusBadRequest)
		return
	}
	if req.MaxPages <= 0 {
		req.MaxPages = 1
	}
	if err := h.importUC.Execute(r.Context(), req.SearchTerms, req.MaxPages); err != nil {
		http.Error(w, "failed to trigger import", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "import triggered"})
}
