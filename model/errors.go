package model

import (
	"errors"

	"github.com/deluan/rest"
)

var (
	// Same value as rest.ErrNotFound, so errors.Is matches either one and REST endpoints respond 404.
	ErrNotFound            = rest.ErrNotFound
	ErrInvalidAuth         = errors.New("invalid authentication")
	ErrNotAuthorized       = errors.New("not authorized")
	ErrExpired             = errors.New("access expired")
	ErrNotAvailable        = errors.New("functionality not available")
	ErrValidation          = errors.New("validation error")
	ErrPlaylistNotEditable = errors.New("playlist tracks are not editable")
)
