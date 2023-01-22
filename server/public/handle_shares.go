package public

import (
	"errors"
	"net/http"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
	"github.com/navidrome/navidrome/ui"
)

func (p *Router) handleShares(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get(":id")
	if id == "" {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	// If requested file is a UI asset, just serve it
	_, err := ui.BuildAssets().Open(id)
	if err == nil {
		p.assetsHandler.ServeHTTP(w, r)
		return
	}

	// If it is not, consider it a share ID
	s, err := p.share.Load(r.Context(), id)
	switch {
	case errors.Is(err, model.ErrNotFound):
		log.Error(r, "Share not found", "id", id, err)
		http.Error(w, "Share not found", http.StatusNotFound)
	case err != nil:
		log.Error(r, "Error retrieving share", "id", id, err)
		http.Error(w, "Error retrieving share", http.StatusInternalServerError)
	}
	if err != nil {
		return
	}

	s = p.mapShareInfo(*s)
	server.IndexWithShare(p.ds, ui.BuildAssets(), s)(w, r)
}

func (p *Router) mapShareInfo(s model.Share) *model.Share {
	mapped := &model.Share{
		Description: s.Description,
		Tracks:      s.Tracks,
	}
	for i := range s.Tracks {
		mapped.Tracks[i].ID = encodeMediafileShare(s, s.Tracks[i].ID)
	}
	return mapped
}
