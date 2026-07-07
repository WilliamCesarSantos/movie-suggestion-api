package postgres

import (
	"context"
	"errors"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/repository"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/postgres/model"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type authUserRepository struct {
	db *gorm.DB
}

func NewAuthUserRepository(db *gorm.DB) repository.AuthUserRepository {
	return &authUserRepository{db: db}
}

func (r *authUserRepository) Create(ctx context.Context, user *entity.AuthUser) error {
	logger := log.Ctx(ctx).With().Str("logger", "repo.postgres.auth_user").Logger()
	logger.Info().Str("userId", user.ID).Str("email", user.Email).Msg("creating auth user")

	m := model.AuthUserModel{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Password:  user.Password,
		Roles:     pq.StringArray(user.Roles),
		CreatedAt: user.CreatedAt,
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			logger.Warn().Str("email", user.Email).Msg("auth user already exists")
			return entity.ErrEmailAlreadyExists
		}
		logger.Error().Err(err).Str("userId", user.ID).Str("email", user.Email).Msg("failed to create auth user")
		return err
	}
	logger.Info().Str("userId", user.ID).Str("email", user.Email).Msg("auth user created")
	return nil
}

func (r *authUserRepository) FindByID(ctx context.Context, id string) (*entity.AuthUser, error) {
	logger := log.Ctx(ctx).With().Str("logger", "repo.postgres.auth_user").Logger()
	logger.Info().Str("userId", id).Msg("finding auth user by id")

	var m model.AuthUserModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn().Str("userId", id).Msg("auth user not found by id")
			return nil, entity.ErrAuthUserNotFound
		}
		logger.Error().Err(err).Str("userId", id).Msg("failed to find auth user by id")
		return nil, err
	}
	logger.Info().Str("userId", id).Str("email", m.Email).Msg("auth user found by id")
	return &entity.AuthUser{
		ID:        m.ID,
		Name:      m.Name,
		Email:     m.Email,
		Password:  m.Password,
		Roles:     []string(m.Roles),
		CreatedAt: m.CreatedAt,
	}, nil
}

func (r *authUserRepository) FindByEmail(ctx context.Context, email string) (*entity.AuthUser, error) {
	logger := log.Ctx(ctx).With().Str("logger", "repo.postgres.auth_user").Logger()
	logger.Info().Str("email", email).Msg("finding auth user by email")

	var m model.AuthUserModel
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn().Str("email", email).Msg("auth user not found by email")
			return nil, entity.ErrAuthUserNotFound
		}
		logger.Error().Err(err).Str("email", email).Msg("failed to find auth user by email")
		return nil, err
	}
	logger.Info().Str("userId", m.ID).Str("email", email).Msg("auth user found by email")
	return &entity.AuthUser{
		ID:        m.ID,
		Name:      m.Name,
		Email:     m.Email,
		Password:  m.Password,
		Roles:     []string(m.Roles),
		CreatedAt: m.CreatedAt,
	}, nil
}

func (r *authUserRepository) List(ctx context.Context, filters repository.AuthUserFilters) ([]*entity.AuthUser, int, error) {
	logger := log.Ctx(ctx).With().Str("logger", "repo.postgres.auth_user").Logger()
	logger.Info().Interface("filters", filters).Msg("listing auth users")

	query := r.db.WithContext(ctx).Model(&model.AuthUserModel{})
	if filters.Email != "" {
		query = query.Where("email = ?", filters.Email)
	}
	if filters.Name != "" {
		query = query.Where("name ILIKE ?", "%"+filters.Name+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		logger.Error().Err(err).Interface("filters", filters).Msg("failed to count auth users")
		return nil, 0, err
	}

	page := filters.Page
	if page < 1 {
		page = 1
	}
	pageSize := filters.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var models []model.AuthUserModel
	offset := (page - 1) * pageSize
	if err := query.Order("created_at ASC").Offset(offset).Limit(pageSize).Find(&models).Error; err != nil {
		logger.Error().Err(err).Interface("filters", filters).Msg("failed to list auth users")
		return nil, 0, err
	}

	users := make([]*entity.AuthUser, len(models))
	for i, m := range models {
		users[i] = &entity.AuthUser{
			ID:        m.ID,
			Name:      m.Name,
			Email:     m.Email,
			Roles:     []string(m.Roles),
			CreatedAt: m.CreatedAt,
		}
	}
	logger.Info().Int("total", int(total)).Int("returned", len(users)).Msg("auth users listed")
	return users, int(total), nil
}

func (r *authUserRepository) Update(ctx context.Context, id string, update repository.AuthUserUpdate) error {
	logger := log.Ctx(ctx).With().Str("logger", "repo.postgres.auth_user").Logger()
	logger.Info().Str("userId", id).Bool("nameChanged", update.Name != nil).Bool("passwordChanged", update.Password != nil).Bool("rolesChanged", update.Roles != nil).Msg("updating auth user")

	updates := map[string]any{}
	if update.Name != nil {
		updates["name"] = *update.Name
	}
	if update.Password != nil {
		updates["password"] = *update.Password
	}
	if update.Roles != nil {
		updates["roles"] = pq.StringArray(*update.Roles)
	}

	if len(updates) == 0 {
		return nil
	}

	res := r.db.WithContext(ctx).Model(&model.AuthUserModel{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		logger.Error().Err(res.Error).Str("userId", id).Msg("failed to update auth user")
		return res.Error
	}
	if res.RowsAffected == 0 {
		logger.Warn().Str("userId", id).Msg("auth user not found during update")
		return entity.ErrAuthUserNotFound
	}
	logger.Info().Str("userId", id).Msg("auth user updated")
	return nil
}
