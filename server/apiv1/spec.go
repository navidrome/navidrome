package apiv1

import (
	"fmt"
	"net/http"

	"github.com/navidrome/navidrome/utils/req"
	"github.com/zeebo/xxh3"
)

func specHandler(body []byte, contentType string) http.HandlerFunc {
	etag := fmt.Sprintf("%016x", xxh3.Hash(body))
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
