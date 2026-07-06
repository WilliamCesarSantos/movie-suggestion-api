package entity

import (
	"errors"
	"time"
)

var ErrAuthUserNotFound = errors.New("auth user not found")
var ErrEmailAlreadyExists = errors.New("email already exists")
var ErrInvalidUserPatchInput = errors.New("invalid user patch input")
var ErrUserPatchForbidden = errors.New("user patch forbidden")

type AuthUser struct {
	ID        string
	Name      string
	Email     string
	Password  string
	Roles     []string
	CreatedAt time.Time
}
