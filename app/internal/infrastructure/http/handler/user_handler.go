package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/repository"
	domainusecase "github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/usecase"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/auth"
	cursorinfra "github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/cursor"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/http/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type UserHandler struct {
	manageUserUC           domainusecase.ManageUserUseCase
	recommendUC            domainusecase.RecommendMoviesUseCase
	listUsersUC            domainusecase.ListUsersUseCase
	patchUserUC            domainusecase.PatchUserUseCase
	authUserRepo           repository.AuthUserRepository
	passwordService        *auth.PasswordService
	cursorSecret           string
	recommendationMaxLimit int
}

func NewUserHandler(
	manageUserUC domainusecase.ManageUserUseCase,
	recommendUC domainusecase.RecommendMoviesUseCase,
	listUsersUC domainusecase.ListUsersUseCase,
	patchUserUC domainusecase.PatchUserUseCase,
	authUserRepo repository.AuthUserRepository,
	passwordService *auth.PasswordService,
	cursorSecret string,
	recommendationMaxLimit int,
) *UserHandler {
	return &UserHandler{
		manageUserUC:           manageUserUC,
		recommendUC:            recommendUC,
		listUsersUC:            listUsersUC,
		patchUserUC:            patchUserUC,
		authUserRepo:           authUserRepo,
		passwordService:        passwordService,
		cursorSecret:           cursorSecret,
		recommendationMaxLimit: recommendationMaxLimit,
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
	logger := log.Ctx(r.Context()).With().Str("logger", "http.user_handler").Logger()
	logger.Info().Msg("create user request received")

	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn().Msg("create user rejected: invalid body")
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Email == "" || req.Password == "" {
		logger.Warn().Str("email", req.Email).Msg("create user rejected: missing required fields")
		http.Error(w, "name, email and password required", http.StatusBadRequest)
		return
	}
	if len(req.Roles) == 0 {
		logger.Warn().Str("email", req.Email).Msg("create user rejected: roles required")
		http.Error(w, "roles required", http.StatusBadRequest)
		return
	}

	userID := uuid.New().String()
	hashedPassword, err := h.passwordService.Hash(req.Password)
	if err != nil {
		logger.Error().Err(err).Str("email", req.Email).Msg("failed to hash password")
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
			logger.Warn().Str("email", req.Email).Msg("create user rejected: email already exists")
			http.Error(w, "email already exists", http.StatusConflict)
			return
		}
		logger.Error().Err(err).Str("email", req.Email).Msg("failed to create auth user")
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
		logger.Error().Err(err).Str("userId", userID).Str("email", req.Email).Msg("failed to create graph user")
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}
	logger.Info().Str("userId", userID).Str("email", req.Email).Msg("user created")

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
	logger := log.Ctx(r.Context()).With().Str("logger", "http.user_handler").Logger()
	logger.Info().Str("userId", id).Msg("get user request received")

	user, err := h.manageUserUC.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) {
			logger.Warn().Str("userId", id).Msg("user not found")
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		logger.Error().Err(err).Str("userId", id).Msg("failed to get user")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	logger.Info().Str("userId", id).Msg("user returned")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(user)
}

type listUsersItem struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Email     string   `json:"email"`
	Roles     []string `json:"roles"`
	CreatedAt string   `json:"createdAt"`
}

type listUsersResponse struct {
	Data     []listUsersItem `json:"data"`
	Total    int             `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"pageSize"`
}

type patchUserRequest struct {
	Name     *string   `json:"name"`
	Password *string   `json:"password"`
	Roles    *[]string `json:"roles"`
}

type patchUserResponse struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Email     string   `json:"email"`
	Roles     []string `json:"roles"`
	CreatedAt string   `json:"createdAt"`
}

func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	callerEmail, _ := r.Context().Value(middleware.ContextKeyUserEmail).(string)
	roles, _ := r.Context().Value(middleware.ContextKeyRoles).([]string)
	logger := log.Ctx(r.Context()).With().Str("logger", "http.user_handler").Logger()
	logger.Info().Str("callerEmail", callerEmail).Msg("list users request received")

	callerHasWrite := false
	for _, role := range roles {
		if role == "*" || role == "users:write" {
			callerHasWrite = true
			break
		}
	}

	input := domainusecase.ListUsersInput{
		Email: r.URL.Query().Get("email"),
		Name:  r.URL.Query().Get("name"),
	}

	if v := r.URL.Query().Get("page"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 {
			logger.Warn().Str("page", v).Msg("list users rejected: invalid page")
			http.Error(w, "invalid page parameter", http.StatusBadRequest)
			return
		}
		input.Page = p
	}
	if v := r.URL.Query().Get("pageSize"); v != "" {
		ps, err := strconv.Atoi(v)
		if err != nil || ps < 1 || ps > 100 {
			logger.Warn().Str("pageSize", v).Msg("list users rejected: invalid pageSize")
			http.Error(w, "invalid pageSize parameter (must be 1-100)", http.StatusBadRequest)
			return
		}
		input.PageSize = ps
	}

	result, err := h.listUsersUC.Execute(r.Context(), callerEmail, callerHasWrite, input)
	if err != nil {
		logger.Error().Err(err).Str("callerEmail", callerEmail).Msg("failed to list users")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	logger.Info().Int("total", result.Total).Int("page", result.Page).Int("pageSize", result.PageSize).Int("returned", len(result.Users)).Msg("users listed")

	items := make([]listUsersItem, len(result.Users))
	for i, u := range result.Users {
		items[i] = listUsersItem{
			ID:        u.ID,
			Name:      u.Name,
			Email:     u.Email,
			Roles:     u.Roles,
			CreatedAt: u.CreatedAt.Format(time.RFC3339),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(listUsersResponse{
		Data:     items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

func (h *UserHandler) PatchUser(w http.ResponseWriter, r *http.Request) {
	if h.patchUserUC == nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	logger := log.Ctx(r.Context()).With().Str("logger", "http.user_handler").Logger()
	logger.Info().Msg("patch user request received")

	var req patchUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn().Msg("patch user rejected: invalid body")
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	callerUserID, _ := r.Context().Value(middleware.ContextKeyUserID).(string)
	targetUserID := chi.URLParam(r, "id")

	out, err := h.patchUserUC.Execute(r.Context(), domainusecase.PatchUserInput{
		TargetUserID: targetUserID,
		CallerUserID: callerUserID,
		Name:         req.Name,
		Password:     req.Password,
		Roles:        req.Roles,
	})
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrInvalidUserPatchInput):
			logger.Warn().Str("targetUserId", targetUserID).Str("callerUserId", callerUserID).Msg("patch user rejected: invalid input")
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		case errors.Is(err, entity.ErrUserPatchForbidden):
			logger.Warn().Str("targetUserId", targetUserID).Str("callerUserId", callerUserID).Msg("patch user rejected: forbidden")
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		case errors.Is(err, entity.ErrAuthUserNotFound):
			logger.Warn().Str("targetUserId", targetUserID).Msg("patch user rejected: user not found")
			http.Error(w, "user not found", http.StatusNotFound)
			return
		default:
			logger.Error().Err(err).Str("targetUserId", targetUserID).Str("callerUserId", callerUserID).Msg("patch user failed")
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}
	logger.Info().Str("targetUserId", targetUserID).Str("callerUserId", callerUserID).Msg("patch user completed")

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(patchUserResponse{
		ID:        out.ID,
		Name:      out.Name,
		Email:     out.Email,
		Roles:     out.Roles,
		CreatedAt: out.CreatedAt,
	})
}

func (h *UserHandler) GetRecommendedMovies(w http.ResponseWriter, r *http.Request) {
	email, _ := r.Context().Value(middleware.ContextKeyUserEmail).(string)
	logger := log.Ctx(r.Context()).With().Str("logger", "http.user_handler").Logger()
	logger.Info().Str("email", email).Msg("recommendations request received")
	if email == "" {
		logger.Warn().Msg("recommendations rejected: unauthorized")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil || parsedLimit < 1 || parsedLimit > 50 {
			logger.Warn().Str("limit", limitStr).Msg("recommendations rejected: invalid limit")
			http.Error(w, "invalid limit parameter (must be 1-50)", http.StatusBadRequest)
			return
		}
		limit = parsedLimit
	}

	offset := 0
	cursorToken := r.URL.Query().Get("cursor")
	if cursorToken != "" {
		decodedCursor, err := cursorinfra.Decode(h.cursorSecret, cursorToken)
		if err != nil {
			logger.Warn().Str("cursor", cursorToken).Msg("recommendations rejected: invalid cursor")
			http.Error(w, "invalid cursor", http.StatusBadRequest)
			return
		}
		offset = decodedCursor.Offset
	}

	var algoOverride *entity.RecommendationAlgorithm
	if algoStr := r.URL.Query().Get("algorithm"); algoStr != "" {
		algo := entity.RecommendationAlgorithm(algoStr)
		algoOverride = &algo
	}
	title := r.URL.Query().Get("title")

	movies, err := h.recommendUC.Execute(r.Context(), email, h.recommendationMaxLimit, algoOverride, title)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) {
			logger.Warn().Str("email", email).Msg("recommendations rejected: user not found")
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, entity.ErrAlgorithmNotFound) {
			logger.Warn().Str("email", email).Msg("recommendations rejected: algorithm not found")
			http.Error(w, "algorithm not found", http.StatusBadRequest)
			return
		}
		logger.Error().Err(err).Str("email", email).Msg("recommendations failed")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	logger.Info().Str("email", email).Int("total", len(movies)).Int("limit", limit).Msg("recommendations resolved")

	total := len(movies)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}

	type recommendedMovieItem struct {
		ID         string  `json:"id"`
		Title      string  `json:"title"`
		Year       string  `json:"year"`
		Poster     string  `json:"poster"`
		ImdbRating float64 `json:"imdbRating"`
	}
	slicedMovies := movies[offset:end]
	items := make([]recommendedMovieItem, len(slicedMovies))
	for i, m := range slicedMovies {
		items[i] = recommendedMovieItem{
			ID:         m.ID,
			Title:      m.Title,
			Year:       m.Year,
			Poster:     m.Poster,
			ImdbRating: m.ImdbRating,
		}
	}

	hasNext := end < total
	hasPrev := offset > 0

	var nextCursor *string
	if hasNext {
		encoded := cursorinfra.Encode(h.cursorSecret, cursorinfra.Cursor{Offset: end, Total: total})
		nextCursor = &encoded
	}

	var prevCursor *string
	if hasPrev {
		prevOffset := offset - limit
		if prevOffset < 0 {
			prevOffset = 0
		}
		encoded := cursorinfra.Encode(h.cursorSecret, cursorinfra.Cursor{Offset: prevOffset, Total: total})
		prevCursor = &encoded
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Data       []recommendedMovieItem `json:"data"`
		NextCursor *string                `json:"nextCursor"`
		PrevCursor *string                `json:"prevCursor"`
		HasNext    bool                   `json:"hasNext"`
		HasPrev    bool                   `json:"hasPrev"`
		Limit      int                    `json:"limit"`
		Count      int                    `json:"count"`
		Total      int                    `json:"total"`
	}{
		Data:       items,
		NextCursor: nextCursor,
		PrevCursor: prevCursor,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
		Limit:      limit,
		Count:      len(items),
		Total:      total,
	})
}
