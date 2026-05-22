package entity

import (
	"errors"
	"time"
)

var ErrAuthUserNotFound = errors.New("auth user not found")
var ErrEmailAlreadyExists = errors.New("email already exists")

type AuthUser struct {
	ID        string
	Name      string
	Email     string
	Password  string
	Roles     []string
	CreatedAt time.Time
}
