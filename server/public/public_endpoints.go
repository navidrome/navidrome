package public

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
	"github.com/navidrome/navidrome/utils"
)

type Router struct {
	http.Handler
	artwork artwork.Artwork
}

func New(artwork artwork.Artwork) *Router {
	p := &Router{artwork: artwork}
	p.Handler = p.routes()

	return p
}

func (p *Router) routes() http.Handler {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(server.URLParamsMiddleware)
		r.Get("/img/{id}", p.handleImages)
	})
	return r
}

func (p *Router) handleImages(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	id := r.URL.Query().Get(":id")
	if id == "" {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	artId, err := artwork.DecodeArtworkID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	size := utils.ParamInt(r, "size", 0)
	imgReader, lastUpdate, err := p.artwork.Get(ctx, artId.String(), size)

	switch {
	case errors.Is(err, context.Canceled):
		return
	case errors.Is(err, model.ErrNotFound):
		log.Error(r, "Couldn't find coverArt", "id", id, err)
		http.Error(w, "Artwork not found", http.StatusNotFound)
		return
	case err != nil:
		log.Error(r, "Error retrieving coverArt", "id", id, err)
		http.Error(w, "Error retrieving coverArt", http.StatusInternalServerError)
		return
	}

	defer imgReader.Close()
	w.Header().Set("Cache-Control", "public, max-age=315360000")
	w.Header().Set("Last-Modified", lastUpdate.Format(time.RFC1123))
	cnt, err := io.Copy(w, imgReader)
	if err != nil {
		log.Warn(ctx, "Error sending image", "count", cnt, err)
	}
}
