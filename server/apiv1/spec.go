package apiv1

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	"github.com/navidrome/navidrome/utils/req"
)

func specHandler(body []byte, contentType string) http.HandlerFunc {
	sum := sha256.Sum256(body)
	etag := hex.EncodeToString(sum[:8])
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"`+etag+`"`)
		w.Header().Set("Cache-Control", "no-cache")
		if req.IfNoneMatch(r, etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}
