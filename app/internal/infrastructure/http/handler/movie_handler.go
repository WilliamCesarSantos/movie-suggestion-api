package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
	domainusecase "github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/usecase"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/http/middleware"
	"github.com/go-chi/chi/v5"
)

type MovieHandler struct {
	getMovieUC   domainusecase.GetMovieUseCase
	manageUserUC domainusecase.ManageUserUseCase
}

func NewMovieHandler(getMovieUC domainusecase.GetMovieUseCase, manageUserUC domainusecase.ManageUserUseCase) *MovieHandler {
	return &MovieHandler{getMovieUC: getMovieUC, manageUserUC: manageUserUC}
}

func (h *MovieHandler) GetMovie(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	movie, err := h.getMovieUC.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, entity.ErrMovieNotFound) {
			http.Error(w, "movie not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(movie)
}

type recordWatchedRequest struct {
	Rating   float64 `json:"rating"`
	Reaction string  `json:"reaction"`
}

type watchedResponse struct {
	UserID    string    `json:"userId"`
	MovieID   string    `json:"movieId"`
	Rating    float64   `json:"rating"`
	Reaction  string    `json:"reaction"`
	WatchedAt time.Time `json:"watchedAt"`
}

func (h *MovieHandler) RecordWatched(w http.ResponseWriter, r *http.Request) {
	movieID := chi.URLParam(r, "id")
	userID, _ := r.Context().Value(middleware.ContextKeyUserID).(string)

	var req recordWatchedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if _, err := h.getMovieUC.GetByID(r.Context(), movieID); err != nil {
		if errors.Is(err, entity.ErrMovieNotFound) {
			http.Error(w, "movie not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	_, err := h.manageUserUC.RecordWatched(r.Context(), userID, movieID, req.Rating, req.Reaction)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) || errors.Is(err, entity.ErrMovieNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(watchedResponse{
		UserID:    userID,
		MovieID:   movieID,
		Rating:    req.Rating,
		Reaction:  req.Reaction,
		WatchedAt: time.Now(),
	})
}
