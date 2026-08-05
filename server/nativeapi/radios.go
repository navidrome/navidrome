package nativeapi

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/consts"
	coreradio "github.com/navidrome/navidrome/core/radio"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server"
)

func (api *Router) addRadioRoute(r chi.Router) {
	constructor := func(ctx context.Context) rest.Repository {
		return api.ds.Resource(ctx, model.Radio{})
	}
	r.Route("/radio", func(r chi.Router) {
		r.Get("/", rest.GetAll(constructor))
		r.Post("/", rest.Post(constructor))
		r.Route("/{id}", func(r chi.Router) {
			r.Use(server.URLParamsMiddleware)
			r.Get("/", rest.Get(constructor))
			r.Put("/", rest.Put(constructor))
			r.Delete("/", rest.Delete(constructor))
			r.Post("/image", api.uploadRadioImage())
			r.Delete("/image", api.deleteRadioImage())
			r.Post("/nowplaying", api.startRadioNowPlaying())
			r.Delete("/nowplaying", api.stopRadioNowPlaying())
		})
	})
}

func (api *Router) startRadioNowPlaying() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if api.radioMetadata == nil {
			http.Error(w, "radio metadata is unavailable", http.StatusServiceUnavailable)
			return
		}

		radioID := chi.URLParam(r, "id")
		radio, err := api.ds.Radio(r.Context()).Get(radioID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = api.radioMetadata.Start(r.Context(), radioNowPlayingSessionID(r.Context()), coreradio.Station{
			ID:        radio.ID,
			StreamURL: radio.StreamUrl,
		})
		if err != nil {
			if errors.Is(err, coreradio.ErrInvalidSession) || errors.Is(err, coreradio.ErrInvalidStation) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func (api *Router) stopRadioNowPlaying() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if api.radioMetadata == nil {
			http.Error(w, "radio metadata is unavailable", http.StatusServiceUnavailable)
			return
		}

		api.radioMetadata.Stop(radioNowPlayingSessionID(r.Context()))
		w.WriteHeader(http.StatusNoContent)
	}
}

func radioNowPlayingSessionID(ctx context.Context) string {
	user, hasUser := request.UserFrom(ctx)
	clientID, _ := request.ClientUniqueIdFrom(ctx)
	if clientID == "" {
		clientID = "default"
	}
	if hasUser && user.ID != "" {
		return user.ID + ":" + clientID
	}
	if username, ok := request.UsernameFrom(ctx); ok && username != "" {
		return username + ":" + clientID
	}
	return clientID
}

func (api *Router) uploadRadioImage() http.HandlerFunc {
	return handleImageUpload(func(ctx context.Context, reader io.Reader, ext string) error {
		radioID := chi.URLParamFromCtx(ctx, "id")
		radio, err := api.ds.Radio(ctx).Get(radioID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return model.ErrNotFound
			}
			return err
		}
		oldPath := radio.UploadedImagePath()
		filename, err := api.imgUpload.SetImage(ctx, consts.EntityRadio, radio.ID, radio.Name, oldPath, reader, ext)
		if err != nil {
			return err
		}
		radio.UploadedImage = filename
		return api.ds.Radio(ctx).Put(radio, "UploadedImage")
	})
}

func (api *Router) deleteRadioImage() http.HandlerFunc {
	return handleImageDelete(func(ctx context.Context) error {
		radioID := chi.URLParamFromCtx(ctx, "id")
		radio, err := api.ds.Radio(ctx).Get(radioID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return model.ErrNotFound
			}
			return err
		}
		if err := api.imgUpload.RemoveImage(ctx, radio.UploadedImagePath()); err != nil {
			return err
		}
		radio.UploadedImage = ""
		return api.ds.Radio(ctx).Put(radio, "UploadedImage")
	})
}
