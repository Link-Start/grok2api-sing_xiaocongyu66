package repository

import "errors"

var (
	ErrNotFound      = errors.New("repository: not found")
	ErrConflict      = errors.New("repository: conflict")
	ErrLimitExceeded = errors.New("repository: limit exceeded")
	ErrInvalidInput  = errors.New("repository: invalid input")
)
