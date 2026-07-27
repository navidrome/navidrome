// Package imghttp holds the HTTP caching contract shared by the subsonic, public, and jellyfin
// image handlers, so they apply identical headers without importing each other.
package imghttp

import (
	"net/http"
	"strings"

	"github.com/navidrome/navidrome/core/artwork"
)

// WriteImageHeaders applies the artwork caching contract and reports whether a 304 was written
// (the caller must then not write a body). requestedHash is the hash the client asserted, or "".
func WriteImageHeaders(w http.ResponseWriter, r *http.Request, img *artwork.Image, requestedHash string) (wrote304 bool) {
	h := w.Header()
	// Placeholders are transient stand-ins for unresolved art: never cached, no validators.
	if img.Placeholder {
		h.Set("Cache-Control", "no-store")
		return false
	}

	// The ETag versions the served representation, so a re-encoding config change invalidates a
	// revalidating client's cache; full-size originals are their own pixel hash.
	etag := img.ETag
	if etag == "" {
		etag = img.Hash
	}
	// An empty validator would be shared by every such response, 304ing clients that echo it back.
	if etag != "" {
		h.Set("ETag", `"`+etag+`"`)
	}
	if !img.LastUpdated.IsZero() {
		h.Set("Last-Modified", img.LastUpdated.UTC().Format(http.TimeFormat))
	}
	// Immutable only when the client asked for the current pixel hash; anything else revalidates,
	// so art that gets re-resolved is never pinned.
	if requestedHash != "" && requestedHash == img.Hash {
		h.Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		h.Set("Cache-Control", "public, no-cache")
	}

	if etag != "" && ifNoneMatch(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return true
	}
	return false
}

// ifNoneMatch reports whether If-None-Match asserts hash, using RFC 9110 weak comparison.
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
