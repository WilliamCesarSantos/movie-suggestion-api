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
"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/http/middleware"
"github.com/go-chi/chi/v5"
"github.com/google/uuid"
)

type UserHandler struct {
manageUserUC    domainusecase.ManageUserUseCase
suggestUC       domainusecase.SuggestMoviesUseCase
updateProfileUC domainusecase.UpdateUserProfileUseCase
listUsersUC     domainusecase.ListUsersUseCase
authUserRepo    repository.AuthUserRepository
passwordService *auth.PasswordService
}

func NewUserHandler(
manageUserUC domainusecase.ManageUserUseCase,
suggestUC domainusecase.SuggestMoviesUseCase,
updateProfileUC domainusecase.UpdateUserProfileUseCase,
listUsersUC domainusecase.ListUsersUseCase,
authUserRepo repository.AuthUserRepository,
passwordService *auth.PasswordService,
) *UserHandler {
return &UserHandler{
manageUserUC:    manageUserUC,
suggestUC:       suggestUC,
updateProfileUC: updateProfileUC,
listUsersUC:     listUsersUC,
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

func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
callerEmail, _ := r.Context().Value(middleware.ContextKeyUserEmail).(string)
roles, _ := r.Context().Value(middleware.ContextKeyRoles).([]string)

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
http.Error(w, "invalid page parameter", http.StatusBadRequest)
return
}
input.Page = p
}
if v := r.URL.Query().Get("pageSize"); v != "" {
ps, err := strconv.Atoi(v)
if err != nil || ps < 1 || ps > 100 {
http.Error(w, "invalid pageSize parameter (must be 1-100)", http.StatusBadRequest)
return
}
input.PageSize = ps
}

result, err := h.listUsersUC.Execute(r.Context(), callerEmail, callerHasWrite, input)
if err != nil {
http.Error(w, "internal server error", http.StatusInternalServerError)
return
}

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

func (h *UserHandler) GetSuggestions(w http.ResponseWriter, r *http.Request) {
email, _ := r.Context().Value(middleware.ContextKeyUserEmail).(string)
if email == "" {
http.Error(w, "unauthorized", http.StatusUnauthorized)
return
}

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

movies, err := h.suggestUC.Execute(r.Context(), email, limit, algoOverride)
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

type suggestionItem struct {
ID         string  `json:"id"`
Title      string  `json:"title"`
Year       string  `json:"year"`
Poster     string  `json:"poster"`
ImdbRating float64 `json:"imdbRating"`
}
result2 := make([]suggestionItem, len(movies))
for i, m := range movies {
result2[i] = suggestionItem{
ID:         m.ID,
Title:      m.Title,
Year:       m.Year,
Poster:     m.Poster,
ImdbRating: m.ImdbRating,
}
}
w.Header().Set("Content-Type", "application/json")
_ = json.NewEncoder(w).Encode(result2)
}
