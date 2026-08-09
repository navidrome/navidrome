package nativeapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server"
)

func (api *Router) addPodcastRoutes(r chi.Router) {
	channelConstructor := func(ctx context.Context) rest.Repository {
		return api.ds.Resource(ctx, model.PodcastChannel{})
	}
	episodeConstructor := func(ctx context.Context) rest.Repository {
		return api.ds.Resource(ctx, model.PodcastEpisode{})
	}
	r.Route("/podcastChannel", func(r chi.Router) {
		r.Get("/", rest.GetAll(channelConstructor))
		r.Get("/search", api.searchPodcastChannels())
		r.Get("/top", api.topPodcastChannels())
		r.Post("/", api.subscribePodcastChannel())
		r.Route("/{id}", func(r chi.Router) {
			r.Use(server.URLParamsMiddleware)
			r.Get("/", rest.Get(channelConstructor))
			r.Put("/", rest.Put(channelConstructor))
			r.Delete("/", api.unsubscribePodcastChannel())
			r.Put("/subscription", api.updatePodcastSubscription())
			r.Post("/refresh", api.refreshPodcastChannel())
			r.Post("/image", api.uploadPodcastChannelImage())
			r.Delete("/image", api.deletePodcastChannelImage())
		})
	})
	r.Route("/podcastEpisode", func(r chi.Router) {
		r.Get("/", rest.GetAll(episodeConstructor))
		r.Route("/{id}", func(r chi.Router) {
			r.Use(server.URLParamsMiddleware)
			r.Get("/", rest.Get(episodeConstructor))
			r.Delete("/", api.deletePodcastEpisode())
			r.Post("/download", api.downloadPodcastEpisode())
		})
	})
}

func (api *Router) searchPodcastChannels() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		results, err := api.podcasts.SearchFeeds(r.Context(), q)
		if err != nil {
			_ = rest.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		_ = rest.RespondWithJSON(w, http.StatusOK, results)
	}
}

func (api *Router) topPodcastChannels() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		country := r.URL.Query().Get("country")
		results, err := api.podcasts.TopFeeds(r.Context(), country)
		if err != nil {
			_ = rest.RespondWithError(w, http.StatusBadGateway, err.Error())
			return
		}
		_ = rest.RespondWithJSON(w, http.StatusOK, results)
	}
}

// subscribePodcastChannel handles adding a podcast for the current user: finds-or-creates the
// shared channel for the given feed url, then subscribes the caller to it. Any authenticated user
// can call this, not just admins - subscribing is a personal action, unlike direct channel edits.
func (api *Router) subscribePodcastChannel() http.HandlerFunc {
	type payload struct {
		Url string `json:"url"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var p payload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		channel, err := api.podcasts.Subscribe(r.Context(), p.Url)
		if err != nil {
			_ = rest.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		_ = rest.RespondWithJSON(w, http.StatusOK, channel)
	}
}

// unsubscribePodcastChannel removes the current user's own subscription to the channel. If they
// were the last subscriber, Unsubscribe also tears down the now-orphaned shared channel and files.
func (api *Router) unsubscribePodcastChannel() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParamFromCtx(r.Context(), "id")
		if err := api.podcasts.Unsubscribe(r.Context(), id); err != nil {
			if errors.Is(err, model.ErrNotFound) || errors.Is(err, rest.ErrNotFound) {
				_ = rest.RespondWithError(w, http.StatusNotFound, err.Error())
				return
			}
			if errors.Is(err, rest.ErrPermissionDenied) {
				_ = rest.RespondWithError(w, http.StatusForbidden, err.Error())
				return
			}
			_ = rest.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		// react-admin's bulk delete reads response.json.id off every DELETE
		// response; an empty body resolves to a null/undefined json and crashes.
		_ = rest.RespondWithJSON(w, http.StatusOK, map[string]string{"id": id})
	}
}

// updatePodcastSubscription saves the current user's own download policy/retention settings for
// a channel they're subscribed to - the per-user counterpart to the old channel-level PUT, which
// used to carry these same fields before they moved to PodcastSubscription.
func (api *Router) updatePodcastSubscription() http.HandlerFunc {
	type payload struct {
		DownloadPolicy model.PodcastDownloadPolicy `json:"downloadPolicy"`
		RetentionCount int                         `json:"retentionCount"`
		RetentionDays  int                         `json:"retentionDays"`
		MaxStorageMB   int                         `json:"maxStorageMb"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		channelID := chi.URLParamFromCtx(r.Context(), "id")
		user, ok := request.UserFrom(r.Context())
		if !ok {
			_ = rest.RespondWithError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		existing, err := api.ds.PodcastSubscription(r.Context()).FindByChannelAndUser(channelID, user.ID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				_ = rest.RespondWithError(w, http.StatusNotFound, "not subscribed to this channel")
				return
			}
			_ = rest.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		var p payload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		sub := *existing
		sub.DownloadPolicy = p.DownloadPolicy
		sub.RetentionCount = p.RetentionCount
		sub.RetentionDays = p.RetentionDays
		sub.MaxStorageMB = p.MaxStorageMB
		if err := api.podcasts.UpdateSubscription(r.Context(), &sub); err != nil {
			_ = rest.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		_ = rest.RespondWithJSON(w, http.StatusOK, sub)
	}
}

func (api *Router) refreshPodcastChannel() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParamFromCtx(r.Context(), "id")
		if err := api.podcasts.RefreshChannel(r.Context(), id); err != nil {
			_ = rest.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func (api *Router) deletePodcastEpisode() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParamFromCtx(r.Context(), "id")
		if err := api.podcasts.DeleteEpisode(r.Context(), id); err != nil {
			if errors.Is(err, model.ErrNotFound) || errors.Is(err, rest.ErrNotFound) {
				_ = rest.RespondWithError(w, http.StatusNotFound, err.Error())
				return
			}
			_ = rest.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		_ = rest.RespondWithJSON(w, http.StatusOK, map[string]string{"id": id})
	}
}

// downloadPodcastEpisode enqueues the download and returns immediately,
// detached from the request context so it survives the response being
// sent; the web UI learns about completion via the podcastEpisode SSE
// refresh/status events.
func (api *Router) downloadPodcastEpisode() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParamFromCtx(r.Context(), "id")
		bgCtx := context.WithoutCancel(r.Context())
		go func() {
			if err := api.podcasts.DownloadEpisode(bgCtx, id); err != nil {
				log.Error(bgCtx, "Error downloading podcast episode", "id", id, err)
			}
		}()
		w.WriteHeader(http.StatusOK)
	}
}

func (api *Router) uploadPodcastChannelImage() http.HandlerFunc {
	return handleImageUpload(func(ctx context.Context, reader io.Reader, ext string) error {
		channelID := chi.URLParamFromCtx(ctx, "id")
		channel, err := api.ds.PodcastChannel(ctx).Get(channelID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return model.ErrNotFound
			}
			return err
		}
		oldPath := channel.UploadedImagePath()
		filename, err := api.imgUpload.SetImage(ctx, consts.EntityPodcastChannel, channel.ID, channel.Title, oldPath, reader, ext)
		if err != nil {
			return err
		}
		channel.UploadedImage = filename
		return api.ds.PodcastChannel(ctx).Put(channel, "UploadedImage")
	})
}

func (api *Router) deletePodcastChannelImage() http.HandlerFunc {
	return handleImageDelete(func(ctx context.Context) error {
		channelID := chi.URLParamFromCtx(ctx, "id")
		channel, err := api.ds.PodcastChannel(ctx).Get(channelID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return model.ErrNotFound
			}
			return err
		}
		if err := api.imgUpload.RemoveImage(ctx, channel.UploadedImagePath()); err != nil {
			return err
		}
		channel.UploadedImage = ""
		return api.ds.PodcastChannel(ctx).Put(channel, "UploadedImage")
	})
}
