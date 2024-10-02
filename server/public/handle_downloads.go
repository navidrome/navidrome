package public

import (
	"net/http"

	"github.com/navidrome/navidrome/utils/req"
)

func (pub *Router) handleDownloads(w http.ResponseWriter, r *http.Request) {
	id, err := req.Params(r).String(":id")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = pub.archiver.ZipShare(r.Context(), id, w)
	checkShareError(r.Context(), w, err, id)
}
