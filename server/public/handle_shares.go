package public

import (
	"context"
	"errors"
	"net/http"
	"path"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
	"github.com/navidrome/navidrome/ui"
	"github.com/navidrome/navidrome/utils/req"
)

func (pub *Router) handleShares(w http.ResponseWriter, r *http.Request) {
	id, err := req.Params(r).String(":id")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// If requested file is a UI asset, just serve it
	_, err = ui.BuildAssets().Open(id)
	if err == nil {
		pub.assetsHandler.ServeHTTP(w, r)
		return
	}

	// If it is not, consider it a share ID
	s, err := pub.share.Load(r.Context(), id)
	if err != nil {
		checkShareError(r.Context(), w, err, id)
		return
	}

	s = pub.mapShareInfo(r, *s)
	server.IndexWithShare(pub.ds, ui.BuildAssets(), s)(w, r)
}

func (pub *Router) handleM3U(w http.ResponseWriter, r *http.Request) {
	id, err := req.Params(r).String(":id")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// If it is not, consider it a share ID
	s, err := pub.share.Load(r.Context(), id)
	if err != nil {
		checkShareError(r.Context(), w, err, id)
		return
	}

	s = pub.mapShareToM3U(r, *s)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "audio/x-mpegurl")
	_, _ = w.Write([]byte(s.ToM3U8()))
}

func checkShareError(ctx context.Context, w http.ResponseWriter, err error, id string) {
	switch {
	case errors.Is(err, model.ErrExpired):
		log.Error(ctx, "Share expired", "id", id, err)
		http.Error(w, "Share not available anymore", http.StatusGone)
	case errors.Is(err, model.ErrNotFound):
		log.Error(ctx, "Share not found", "id", id, err)
		http.Error(w, "Share not found", http.StatusNotFound)
	case errors.Is(err, model.ErrNotAuthorized):
		log.Error(ctx, "Share is not downloadable", "id", id, err)
		http.Error(w, "This share is not downloadable", http.StatusForbidden)
	case err != nil:
		log.Error(ctx, "Error retrieving share", "id", id, err)
		http.Error(w, "Error retrieving share", http.StatusInternalServerError)
	}
}

func (pub *Router) mapShareInfo(r *http.Request, s model.Share) *model.Share {
	s.URL = ShareURL(r, s.ID)
	s.ImageURL = ImageURL(r, s.CoverArtID(), consts.UICoverArtSize)
	for i := range s.Tracks {
		s.Tracks[i].ID = encodeMediafileShare(s, s.Tracks[i].ID)
	}
	return &s
}

func (pub *Router) mapShareToM3U(r *http.Request, s model.Share) *model.Share {
	for i := range s.Tracks {
		id := encodeMediafileShare(s, s.Tracks[i].ID)
		s.Tracks[i].Path = publicURL(r, path.Join(consts.URLPathPublic, "s", id), nil)
	}
	return &s
}
