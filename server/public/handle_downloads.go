package public

import (
	"cmp"
	"net/http"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/req"
	"github.com/navidrome/navidrome/utils/str"
)

func (pub *Router) handleDownloads(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := req.Params(r).String(":id")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Load the share before streaming: once ZipShare writes its first byte the
	// status is locked at 200, so errors could no longer be reported.
	s, err := pub.share.Load(ctx, id)
	if err != nil {
		checkShareError(ctx, w, err, id)
		return
	}
	if !s.Downloadable {
		checkShareError(ctx, w, model.ErrNotAuthorized, id)
		return
	}

	name := cmp.Or(s.Description, s.ID)
	w.Header().Set("Content-Disposition", str.ContentDispositionAttachment(name+".zip"))
	w.Header().Set("Content-Type", "application/zip")

	err = pub.archiver.ZipShare(ctx, s, w)
	checkShareError(ctx, w, err, id)
}
