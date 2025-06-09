package api

import "errors"

var (
	ErrNotFound       = errors.New("plugin:not_found")
	ErrNotImplemented = errors.New("plugin:not_implemented")
)
