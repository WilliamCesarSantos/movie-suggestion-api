package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/WilliamCesarSantos/movie-suggestion/internal/domain/entity"
	domainusecase "github.com/WilliamCesarSantos/movie-suggestion/internal/domain/usecase"
	"github.com/go-chi/chi/v5"
)

type UserHandler struct {
	manageUserUC    domainusecase.ManageUserUseCase
	suggestUC       domainusecase.SuggestMoviesUseCase
	updateProfileUC domainusecase.UpdateUserProfileUseCase
}

func NewUserHandler(manageUserUC domainusecase.ManageUserUseCase, suggestUC domainusecase.SuggestMoviesUseCase, updateProfileUC domainusecase.UpdateUserProfileUseCase) *UserHandler {
	return &UserHandler{manageUserUC: manageUserUC, suggestUC: suggestUC, updateProfileUC: updateProfileUC}
}

type createUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	user := &entity.User{Name: req.Name, Email: req.Email}
	if err := h.manageUserUC.Create(r.Context(), user); err != nil {
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user, err := h.manageUserUC.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

type recordWatchedRequest struct {
	MovieID string  `json:"movieId"`
	Rating  float64 `json:"rating"`
}

func (h *UserHandler) RecordWatched(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req recordWatchedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	user, err := h.manageUserUC.RecordWatched(r.Context(), id, req.MovieID, req.Rating)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) || errors.Is(err, entity.ErrMovieNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

type recordLikedRequest struct {
	MovieID   string                     `json:"movieId"`
	Algorithm entity.SuggestionAlgorithm `json:"algorithm"`
}

func (h *UserHandler) RecordLiked(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req recordLikedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	user, err := h.manageUserUC.RecordLiked(r.Context(), id, req.MovieID, req.Algorithm)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) || errors.Is(err, entity.ErrMovieNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

type recordDislikedRequest struct {
	MovieID string `json:"movieId"`
}

func (h *UserHandler) RecordDisliked(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req recordDislikedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	user, err := h.manageUserUC.RecordDisliked(r.Context(), id, req.MovieID)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) || errors.Is(err, entity.ErrMovieNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) GetSuggestions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	limitStr := r.URL.Query().Get("limit")
	limit := 0
	if limitStr != "" {
		limit, _ = strconv.Atoi(limitStr)
	}

	var algoOverride *entity.SuggestionAlgorithm
	if algoStr := r.URL.Query().Get("algorithm"); algoStr != "" {
		algo := entity.SuggestionAlgorithm(algoStr)
		algoOverride = &algo
	}

	movies, err := h.suggestUC.Execute(r.Context(), id, limit, algoOverride)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, entity.ErrAlgorithmNotFound) {
			http.Error(w, "algorithm not found", http.StatusBadRequest)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(movies)
}
