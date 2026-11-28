// Package imghttp holds the HTTP caching contract shared by the subsonic, public, and jellyfin
// image handlers, so they apply identical headers without importing each other.
package imghttp

import (
	"io"
	"io/fs"
	"net/http"
	"strconv"

	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/utils/cache"
	"github.com/navidrome/navidrome/utils/req"
)

const maxBufferedImage = 4 << 20

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

	if etag != "" && req.IfNoneMatch(r, etag) {
		w.WriteHeader(http.StatusNotModified)
		return true
	}
	return false
}

func WriteImage(w http.ResponseWriter, img *artwork.Image) (int64, error) {
	if size := knownSize(img.ReadCloser); size >= 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		n, err := io.Copy(w, img)
		if err != nil && n == 0 {
			withdrawCaching(w)
		}
		return n, err
	}

	head, err := io.ReadAll(io.LimitReader(img, maxBufferedImage+1))
	if err != nil {
		withdrawCaching(w)
		return 0, err
	}
	if len(head) <= maxBufferedImage {
		w.Header().Set("Content-Length", strconv.Itoa(len(head)))
		n, err := w.Write(head)
		return int64(n), err
	}
	n, err := w.Write(head)
	if err != nil {
		return int64(n), err
	}
	rest, err := io.Copy(w, img)
	return int64(n) + rest, err
}

func knownSize(r io.Reader) int64 {
	switch v := r.(type) {
	case *cache.CachedStream:
		if sr, ok := v.Reader.(*io.SectionReader); ok {
			return sr.Size()
		}
	case interface{ Stat() (fs.FileInfo, error) }:
		if fi, err := v.Stat(); err == nil && fi.Mode().IsRegular() {
			return fi.Size()
		}
	}
	return -1
}

func withdrawCaching(w http.ResponseWriter) {
	h := w.Header()
	h.Del("Content-Length")
	h.Del("ETag")
	h.Del("Last-Modified")
	h.Set("Cache-Control", "no-store")
}
