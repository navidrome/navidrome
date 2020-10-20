package model

import "errors"

var (
	ErrNotFound      = errors.New("data not found")
	ErrInvalidAuth   = errors.New("invalid authentication")
	ErrNotAuthorized = errors.New("not authorized")
	ErrNotAvailable  = errors.New("functionality not available")
)
