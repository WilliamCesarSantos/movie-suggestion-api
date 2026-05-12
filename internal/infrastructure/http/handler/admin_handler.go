package handler

import (
	"encoding/json"
	"net/http"

	domainusecase "github.com/WilliamCesarSantos/movie-suggestion/internal/domain/usecase"
)

type AdminHandler struct {
	importUC domainusecase.ImportMoviesUseCase
}

func NewAdminHandler(importUC domainusecase.ImportMoviesUseCase) *AdminHandler {
	return &AdminHandler{importUC: importUC}
}

type triggerImportRequest struct {
	SearchTerms []string `json:"searchTerms"`
	MaxPages    int      `json:"maxPages"`
}

func (h *AdminHandler) TriggerImport(w http.ResponseWriter, r *http.Request) {
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
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "import triggered"})
}
