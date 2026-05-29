package postgres

import (
	"context"
	"errors"

	"github.com/WilliamCesarSantos/movie-suggestion/app/internal/domain/entity"
	"github.com/WilliamCesarSantos/movie-suggestion/app/internal/domain/repository"
	"github.com/WilliamCesarSantos/movie-suggestion/app/internal/infrastructure/postgres/model"
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
