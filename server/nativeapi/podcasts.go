package nativeapi

import (
	"encoding/json"
	"net/http"

	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core/podcasts"
	"github.com/navidrome/navidrome/server"
)

func (api *Router) addPodcastRoute(r chi.Router) {
	repo := api.ds.PodcastChannel()
	r.Route("/podcast", func(r chi.Router) {
		r.Get("/", rest.GetAll(repo))
		r.With(adminOnlyMiddleware).Get("/preview", api.podcastPreview)
		r.Route("/{id}", func(r chi.Router) {
			r.Use(server.URLParamsMiddleware)
			r.Get("/", rest.Get(repo))
			r.Delete("/", rest.Delete(repo))
		})
	})
}

func (api *Router) podcastPreview(w http.ResponseWriter, r *http.Request) {
	feedURL := r.URL.Query().Get("url")
	if feedURL == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}
	feed, err := podcasts.ParseFeedPreview(feedURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	exists, _ := api.ds.PodcastChannel().ExistsByURL(r.Context(), feedURL)
	feed.AlreadyExists = exists
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(feed)
}
