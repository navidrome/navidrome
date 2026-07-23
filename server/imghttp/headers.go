// Package imghttp holds the shared HTTP caching contract for artwork responses, so the
// subsonic, public, and jellyfin image handlers apply identical headers without importing
// each other.
package imghttp

import (
	"net/http"
	"strings"

	"github.com/navidrome/navidrome/core/artwork"
)

// WriteImageHeaders applies the artwork caching contract and reports whether a 304 was written
// (in which case the caller must not write a body). requestedHash is the hash the client asserted
// (id suffix / JWT payload / jellyfin tag param), or "" when the request carried no hash.
func WriteImageHeaders(w http.ResponseWriter, r *http.Request, img *artwork.Image, requestedHash string) (wrote304 bool) {
	h := w.Header()
	// Placeholders are transient stand-ins for not-yet-resolved art: never cached, no validators.
	if img.Placeholder {
		h.Set("Cache-Control", "no-store")
		return false
	}

	// The validator identifies the served representation (resized/re-encoded bytes version it via
	// ETag), so a CoverArtQuality/EnableWebPEncoding change invalidates a revalidating client's
	// cache. Falls back to the pixel hash for full-size originals (bytes == the hash).
	etag := img.ETag
	if etag == "" {
		etag = img.Hash
	}
	h.Set("ETag", `"`+etag+`"`)
	if !img.LastUpdated.IsZero() {
		h.Set("Last-Modified", img.LastUpdated.UTC().Format(http.TimeFormat))
	}
	// Immutable only when the client asked for the exact current pixel hash; bare/legacy/mismatched
	// requests get cheap ETag revalidation instead, which fixes stale art after re-resolution.
	if requestedHash != "" && requestedHash == img.Hash {
		h.Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		h.Set("Cache-Control", "public, no-cache")
	}

	if ifNoneMatch(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return true
	}
	return false
}

// ifNoneMatch reports whether the If-None-Match header asserts the given hash, using weak
// comparison (RFC 9110): "*" matches any current representation and W/ prefixes are ignored.
func ifNoneMatch(header, hash string) bool {
	header = strings.TrimSpace(header)
	if header == "" {
		return false
	}
	if header == "*" {
		return true
	}
	for _, tag := range strings.Split(header, ",") {
		tag = strings.TrimSpace(tag)
		tag = strings.TrimPrefix(tag, "W/")
		if strings.Trim(tag, `"`) == hash {
			return true
		}
	}
	return false
}
