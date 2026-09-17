package apiv1

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"

	"github.com/navidrome/navidrome/api"
)

var (
	specJSONETag = computeETag(api.SpecJSON())
	specYAMLETag = computeETag(api.SpecYAML())
)

func computeETag(body []byte) string {
	sum := sha256.Sum256(body)
	return fmt.Sprintf(`"%x"`, sum[:8])
}

func (rt *Router) serveSpecJSON(w http.ResponseWriter, r *http.Request) {
	serveSpec(w, r, api.SpecJSON(), specJSONETag, "application/json")
}

func (rt *Router) serveSpecYAML(w http.ResponseWriter, r *http.Request) {
	serveSpec(w, r, api.SpecYAML(), specYAMLETag, "application/yaml")
}

func serveSpec(w http.ResponseWriter, r *http.Request, body []byte, etag, contentType string) {
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", contentType)
	if etagMatches(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func etagMatches(ifNoneMatch, etag string) bool {
	for candidate := range strings.SplitSeq(ifNoneMatch, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || strings.TrimPrefix(candidate, "W/") == etag {
			return true
		}
	}
	return false
}
