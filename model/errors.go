package model

import (
	"errors"

	"github.com/deluan/rest"
)

var (
	ErrNotFound            = rest.ErrNotFound
	ErrInvalidAuth         = errors.New("invalid authentication")
	ErrNotAuthorized       = rest.ErrPermissionDenied
	ErrExpired             = errors.New("access expired")
	ErrNotAvailable        = errors.New("functionality not available")
	ErrValidation          = errors.New("validation error")
	ErrPlaylistNotEditable = errors.New("playlist tracks are not editable")
)
