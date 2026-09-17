package apiv1

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"net/http"
	"time"

	"github.com/navidrome/navidrome/api"
)

func (rt *Router) serveSpecJSON(w http.ResponseWriter, r *http.Request) {
	serveSpec(w, r, api.SpecJSON(), "application/json")
}

func (rt *Router) serveSpecYAML(w http.ResponseWriter, r *http.Request) {
	serveSpec(w, r, api.SpecYAML(), "application/yaml")
}

func serveSpec(w http.ResponseWriter, r *http.Request, body []byte, contentType string) {
	sum := sha256.Sum256(body)
	w.Header().Set("ETag", fmt.Sprintf(`"%x"`, sum[:8]))
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", contentType)
	// ServeContent handles If-None-Match/304 and Range; zero modtime disables Last-Modified.
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(body))
}
