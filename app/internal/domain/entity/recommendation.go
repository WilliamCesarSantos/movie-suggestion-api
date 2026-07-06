package entity

import "errors"

var (
	ErrMovieNotFound     = errors.New("movie not found")
	ErrUserNotFound      = errors.New("user not found")
	ErrAlgorithmNotFound = errors.New("algorithm not found")
)
