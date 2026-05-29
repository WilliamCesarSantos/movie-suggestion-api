package model

import (
	"time"

	"github.com/lib/pq"
)

type AuthUserModel struct {
	ID        string         `gorm:"type:uuid;primaryKey"`
	Email     string         `gorm:"uniqueIndex;not null"`
	Name      string         `gorm:"not null"`
	Password  string         `gorm:"not null"`
	Roles     pq.StringArray `gorm:"type:text[];not null"`
	CreatedAt time.Time
}

func (AuthUserModel) TableName() string { return "users" }
