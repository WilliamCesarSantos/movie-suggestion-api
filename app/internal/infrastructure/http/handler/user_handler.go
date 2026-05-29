package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/WilliamCesarSantos/movie-suggestion/app/internal/domain/entity"
	"github.com/WilliamCesarSantos/movie-suggestion/app/internal/domain/repository"
	domainusecase "github.com/WilliamCesarSantos/movie-suggestion/app/internal/domain/usecase"
	"github.com/WilliamCesarSantos/movie-suggestion/app/internal/infrastructure/auth"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type UserHandler struct {
	manageUserUC    domainusecase.ManageUserUseCase
	suggestUC       domainusecase.SuggestMoviesUseCase
	updateProfileUC domainusecase.UpdateUserProfileUseCase
	authUserRepo    repository.AuthUserRepository
	passwordService *auth.PasswordService
}

func NewUserHandler(
	manageUserUC domainusecase.ManageUserUseCase,
	suggestUC domainusecase.SuggestMoviesUseCase,
	updateProfileUC domainusecase.UpdateUserProfileUseCase,
	authUserRepo repository.AuthUserRepository,
	passwordService *auth.PasswordService,
) *UserHandler {
	return &UserHandler{
		manageUserUC:    manageUserUC,
		suggestUC:       suggestUC,
		updateProfileUC: updateProfileUC,
		authUserRepo:    authUserRepo,
		passwordService: passwordService,
	}
}

type createUserRequest struct {
	Name     string   `json:"name"`
	Email    string   `json:"email"`
	Password string   `json:"password"`
	Roles    []string `json:"roles"`
}

type createUserResponse struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Email     string   `json:"email"`
	Roles     []string `json:"roles"`
	CreatedAt string   `json:"createdAt"`
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Email == "" || req.Password == "" {
		http.Error(w, "name, email and password required", http.StatusBadRequest)
		return
	}
	if len(req.Roles) == 0 {
		http.Error(w, "roles required", http.StatusBadRequest)
		return
	}

	userID := uuid.New().String()
	hashedPassword, err := h.passwordService.Hash(req.Password)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	now := time.Now()
	authUser := &entity.AuthUser{
		ID:        userID,
		Name:      req.Name,
		Email:     req.Email,
		Password:  hashedPassword,
		Roles:     req.Roles,
		CreatedAt: now,
	}
	if err := h.authUserRepo.Create(r.Context(), authUser); err != nil {
		if errors.Is(err, entity.ErrEmailAlreadyExists) {
			http.Error(w, "email already exists", http.StatusConflict)
			return
		}
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	neo4jUser := &entity.User{
		ID:        userID,
		Name:      req.Name,
		Email:     req.Email,
		CreatedAt: now,
	}
	if err := h.manageUserUC.Create(r.Context(), neo4jUser); err != nil {
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(createUserResponse{
		ID:        userID,
		Name:      req.Name,
		Email:     req.Email,
		Roles:     req.Roles,
		CreatedAt: now.Format(time.RFC3339),
	})
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
	_ = json.NewEncoder(w).Encode(user)
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
	_ = json.NewEncoder(w).Encode(movies)
}
