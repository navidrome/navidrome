package model

import "errors"

var (
	ErrNotFound      = errors.New("data not found")
	ErrInvalidAuth   = errors.New("invalid authentication")
	ErrNotAuthorized = errors.New("not authorized")
	ErrExpired       = errors.New("access expired")
	ErrNotAvailable  = errors.New("functionality not available")
	ErrValidation    = errors.New("validation error")
)
