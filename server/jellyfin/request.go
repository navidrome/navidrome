package jellyfin

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// decodeJSONBody decodes exactly one JSON value of at most limit bytes into v, answering 413 or 400 itself on failure.
func decodeJSONBody(w http.ResponseWriter, r *http.Request, limit int64, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit))
	err := dec.Decode(v)
	if err == nil {
		// Decode alone would ignore anything after the first value.
		if _, err = dec.Token(); errors.Is(err, io.EOF) {
			return true
		}
	}
	if errors.As(err, new(*http.MaxBytesError)) {
		http.Error(w, "Request Entity Too Large", http.StatusRequestEntityTooLarge)
	} else {
		http.Error(w, "Bad Request", http.StatusBadRequest)
	}
	return false
}
