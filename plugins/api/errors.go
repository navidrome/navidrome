package api

import "errors"

var (
	// ErrNotImplemented indicates that the plugin does not implement the requested method.
	// No logic should be executed by the plugin.
	ErrNotImplemented = errors.New("plugin:not_implemented")

	// ErrNotFound indicates that the requested resource was not found by the plugin.
	ErrNotFound = errors.New("plugin:not_found")
)
