package public

import (
	"net/http"
	"path"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
	"github.com/navidrome/navidrome/ui"
)

type Router struct {
	http.Handler
	artwork       artwork.Artwork
	streamer      core.MediaStreamer
	share         core.Share
	assetsHandler http.Handler
	ds            model.DataStore
}

func New(ds model.DataStore, artwork artwork.Artwork, streamer core.MediaStreamer, share core.Share) *Router {
	p := &Router{ds: ds, artwork: artwork, streamer: streamer, share: share}
	shareRoot := path.Join(conf.Server.BaseURL, consts.URLPathPublic)
	p.assetsHandler = http.StripPrefix(shareRoot, http.FileServer(http.FS(ui.BuildAssets())))
	p.Handler = p.routes()

	return p
}

func (p *Router) routes() http.Handler {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(server.URLParamsMiddleware)
		r.HandleFunc("/img/{id}", p.handleImages)
		if conf.Server.EnableSharing {
			r.HandleFunc("/s/{id}", p.handleStream)
			r.HandleFunc("/{id}", p.handleShares)
			r.HandleFunc("/", p.handleShares)
			r.Handle("/*", p.assetsHandler)
		}
	})
	return r
}

func ShareURL(r *http.Request, id string) string {
	uri := path.Join(consts.URLPathPublic, id)
	return server.AbsoluteURL(r, uri, nil)
}
