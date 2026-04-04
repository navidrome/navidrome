package nativeapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
	"github.com/navidrome/navidrome/server/radiobrowser"
)

func (api *Router) addRadioRoute(r chi.Router) {
	constructor := func(ctx context.Context) rest.Repository {
		return api.ds.Resource(ctx, model.Radio{})
	}
	r.Route("/radio", func(r chi.Router) {
		r.Get("/", rest.GetAll(constructor))
		r.Post("/", rest.Post(constructor))
		r.Get("/browser/search", api.searchRadioBrowser())
		r.Post("/browser/click", api.radioBrowserClick())
		r.Route("/{id}", func(r chi.Router) {
			r.Use(server.URLParamsMiddleware)
			r.Get("/", rest.Get(constructor))
			r.Put("/", rest.Put(constructor))
			r.Delete("/", rest.Delete(constructor))
			r.Post("/image", api.uploadRadioImage())
			r.Delete("/image", api.deleteRadioImage())
		})
	})
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

func (api *Router) searchRadioBrowser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		limit := 0
		if ls := strings.TrimSpace(r.URL.Query().Get("limit")); ls != "" {
			if n, err := strconv.Atoi(ls); err == nil {
				limit = n
			}
		}
		stations, err := radiobrowser.Search(r.Context(), q, limit)
		if err != nil {
			if errors.Is(err, radiobrowser.ErrQueryTooShort) || errors.Is(err, radiobrowser.ErrQueryTooLong) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"stations": stations})
	}
}

func (api *Router) radioBrowserClick() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			StreamURL string `json:"streamUrl"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		streamURL := strings.TrimSpace(body.StreamURL)
		if streamURL == "" {
			http.Error(w, "streamUrl required", http.StatusBadRequest)
			return
		}
go func(u string) {
			defer func() { _ = recover() }()
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			radiobrowser.NotifyClick(ctx, u)
		}(streamURL)
		w.WriteHeader(http.StatusNoContent)
	}
}
