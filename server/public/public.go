package public

import (
	"context"
	"net/http"
	"path"

	"github.com/go-chi/chi/v5"
	"github.com/SunilSherkhane/navidrome/conf"
	"github.com/SunilSherkhane/navidrome/consts"
	"github.com/SunilSherkhane/navidrome/core"
	"github.com/SunilSherkhane/navidrome/core/artwork"
	"github.com/SunilSherkhane/navidrome/core/publicurl"
	"github.com/SunilSherkhane/navidrome/core/stream"
	"github.com/SunilSherkhane/navidrome/model"
	"github.com/SunilSherkhane/navidrome/server"
	"github.com/SunilSherkhane/navidrome/ui"
)

type Router struct {
	http.Handler
	artwork       artwork.Artwork
	streamer      stream.MediaStreamer
	decider       stream.TranscodeDecider
	archiver      core.Archiver
	share         core.Share
	assetsHandler http.Handler
	ds            model.DataStore
}

func New(ds model.DataStore, artwork artwork.Artwork, streamer stream.MediaStreamer, decider stream.TranscodeDecider, share core.Share, archiver core.Archiver) *Router {
	p := &Router{ds: ds, artwork: artwork, streamer: streamer, decider: decider, share: share, archiver: archiver}
	shareRoot := path.Join(conf.Server.BasePath, consts.URLPathPublic)
	p.assetsHandler = http.StripPrefix(shareRoot, http.FileServer(http.FS(ui.BuildAssets())))
	p.Handler = p.routes()

	return p
}

func (pub *Router) routes() http.Handler {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(server.URLParamsMiddleware)
		r.Group(func(r chi.Router) {
			r.Use(server.ThrottleBacklog(conf.Server.DevArtworkMaxRequests, conf.Server.DevArtworkThrottleBacklogLimit,
				conf.Server.DevArtworkThrottleBacklogTimeout))
			r.HandleFunc("/img/{id}", pub.handleImages)
		})
		if conf.Server.EnableSharing {
			r.HandleFunc("/s/{id}", pub.handleStream)
			if conf.Server.EnableDownloads {
				r.HandleFunc("/d/{id}", pub.handleDownloads)
			}
			r.HandleFunc("/{id}/m3u", pub.handleM3U)
			r.HandleFunc("/{id}", pub.handleShares)
			r.HandleFunc("/", pub.handleShares)
			r.Handle("/*", pub.assetsHandler)
		}
	})
	return r
}

func ShareURL(ctx context.Context, id string) string {
	uri := path.Join(consts.URLPathPublic, id)
	return publicurl.PublicURL(ctx, uri, nil)
}
