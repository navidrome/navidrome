package model

import "errors"

var (
	ErrNotFound      = errors.New("data not found")
	ErrInvalidAuth   = errors.New("invalid authentication")
	ErrNotAuthorized = errors.New("not authorized")
	ErrExpired       = errors.New("access expired")
	ErrNotAvailable  = errors.New("functionality not available")
	ErrValidation    = errors.New("validation error")

	// ErrPlaylistNotEditable means the playlist's tracks are managed automatically
	// (smart playlist rules, or a synced file) and cannot be modified by a user.
	ErrPlaylistNotEditable = errors.New("playlist tracks are not editable")
)
