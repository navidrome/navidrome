package server

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// SetWriteTimeout sets a write deadline on the response writer by walking the
// Unwrap chain to find a writer that supports SetWriteDeadline.
func SetWriteTimeout(rw io.Writer, timeout time.Duration) error {
	for {
		switch t := rw.(type) {
		case interface{ SetWriteDeadline(time.Time) error }:
			return t.SetWriteDeadline(time.Now().Add(timeout))
		case interface{ Unwrap() http.ResponseWriter }:
			rw = t.Unwrap()
		default:
			return fmt.Errorf("%T - %w", rw, http.ErrNotSupported)
		}
	}
}
