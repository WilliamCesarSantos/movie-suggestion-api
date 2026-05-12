package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/WilliamCesarSantos/movie-suggestion/internal/domain/entity"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/domain/repository"
	"github.com/go-chi/chi/v5"
)

type MovieHandler struct {
	movieRepo repository.MovieRepository
}

func NewMovieHandler(movieRepo repository.MovieRepository) *MovieHandler {
	return &MovieHandler{movieRepo: movieRepo}
}

func (h *MovieHandler) GetMovie(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	movie, err := h.movieRepo.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, entity.ErrMovieNotFound) {
			http.Error(w, "movie not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(movie)
}
