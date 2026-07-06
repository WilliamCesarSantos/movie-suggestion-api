package postgres

import (
	"context"
	"errors"

	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/entity"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/domain/repository"
	"github.com/WilliamCesarSantos/movie-suggestion-api/app/internal/infrastructure/postgres/model"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type authUserRepository struct {
	db *gorm.DB
}

func NewAuthUserRepository(db *gorm.DB) repository.AuthUserRepository {
	return &authUserRepository{db: db}
}

func (r *authUserRepository) Create(ctx context.Context, user *entity.AuthUser) error {
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
			return entity.ErrEmailAlreadyExists
		}
		return err
	}
	return nil
}

func (r *authUserRepository) FindByID(ctx context.Context, id string) (*entity.AuthUser, error) {
	var m model.AuthUserModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, entity.ErrAuthUserNotFound
		}
		return nil, err
	}
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
	var m model.AuthUserModel
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, entity.ErrAuthUserNotFound
		}
		return nil, err
	}
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
	query := r.db.WithContext(ctx).Model(&model.AuthUserModel{})
	if filters.Email != "" {
		query = query.Where("email = ?", filters.Email)
	}
	if filters.Name != "" {
		query = query.Where("name ILIKE ?", "%"+filters.Name+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
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
	return users, int(total), nil
}

func (r *authUserRepository) Update(ctx context.Context, id string, update repository.AuthUserUpdate) error {
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
		return res.Error
	}
	if res.RowsAffected == 0 {
		return entity.ErrAuthUserNotFound
	}
	return nil
}
