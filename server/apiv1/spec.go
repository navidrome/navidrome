package apiv1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strconv"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
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

// withBasePath adds BasePath to the advertised server URL, since only the running server knows it.
func withBasePath(body []byte, key string, quotedInBundle bool) []byte {
	serverURL := path.Join(conf.Server.BasePath, consts.URLPathAPIv1)
	if serverURL == consts.URLPathAPIv1 {
		return body
	}
	oldURL := consts.URLPathAPIv1
	if quotedInBundle {
		oldURL = strconv.Quote(oldURL)
	}
	old := []byte(key + oldURL)
	if bytes.Count(body, old) != 1 {
		log.Error("API v1: server URL not found in the bundled spec, serving it without the base path", "key", key)
		return body
	}
	// A JSON string is also a valid YAML double-quoted scalar, so one encoding escapes both formats.
	newURL, _ := json.Marshal(serverURL)
	return bytes.Replace(body, old, append([]byte(key), newURL...), 1)
}
