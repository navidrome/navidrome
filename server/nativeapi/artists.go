package nativeapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
)

func (api *Router) addArtistRoute(r chi.Router) {
	constructor := func(ctx context.Context) rest.Repository {
		return api.ds.Resource(ctx, model.Artist{})
	}
	r.Route("/artist", func(r chi.Router) {
		r.Get("/", rest.GetAll(constructor))
		r.Route("/{id}", func(r chi.Router) {
			r.Use(server.URLParamsMiddleware)
			r.Get("/", rest.Get(constructor))
			r.Post("/image", api.uploadArtistImage())
			r.Delete("/image", api.deleteArtistImage())
		})
	})
}

func (api *Router) uploadArtistImage() http.HandlerFunc {
	return handleImageUpload(func(ctx context.Context, reader io.Reader, ext string) error {
		artistID := chi.URLParamFromCtx(ctx, "id")
		ar, err := api.ds.Artist(ctx).Get(artistID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return model.ErrNotFound
			}
			return err
		}
		oldPath := ar.UploadedImagePath()
		filename, err := api.imgUpload.SetImage(ctx, consts.EntityArtist, ar.ID, ar.Name, oldPath, reader, ext)
		if err != nil {
			return err
		}
		ar.UploadedImage = filename
		now := time.Now()
		ar.UpdatedAt = &now
		return api.ds.Artist(ctx).Put(ar, "uploaded_image", "updated_at")
	})
}

func (api *Router) deleteArtistImage() http.HandlerFunc {
	return handleImageDelete(func(ctx context.Context) error {
		artistID := chi.URLParamFromCtx(ctx, "id")
		ar, err := api.ds.Artist(ctx).Get(artistID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return model.ErrNotFound
			}
			return err
		}
		if err := api.imgUpload.RemoveImage(ctx, ar.UploadedImagePath()); err != nil {
			return err
		}
		ar.UploadedImage = ""
		now := time.Now()
		ar.UpdatedAt = &now
		return api.ds.Artist(ctx).Put(ar, "uploaded_image", "updated_at")
	})
}
