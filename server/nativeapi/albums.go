package nativeapi

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
)

func (api *Router) addAlbumRoute(r chi.Router) {
	constructor := func(ctx context.Context) rest.Repository {
		return api.ds.Resource(ctx, model.Album{})
	}
	r.Route("/album", func(r chi.Router) {
		r.Get("/", rest.GetAll(constructor))
		r.Route("/{id}", func(r chi.Router) {
			r.Use(server.URLParamsMiddleware)
			r.Get("/", rest.Get(constructor))
			r.Post("/image", api.uploadAlbumImage())
			r.Delete("/image", api.deleteAlbumImage())
		})
	})
}

func (api *Router) uploadAlbumImage() http.HandlerFunc {
	return handleImageUpload(func(ctx context.Context, reader io.Reader, ext string) error {
		albumID := chi.URLParamFromCtx(ctx, "id")
		al, err := api.ds.Album(ctx).Get(albumID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return model.ErrNotFound
			}
			return err
		}
		filename, err := api.imgUpload.SetImage(ctx, consts.EntityAlbum, al.ID, al.Name, al.UploadedImagePath(), reader, ext)
		if err != nil {
			return err
		}
		return api.ds.Album(ctx).UpdateImage(al.ID, filename)
	})
}

func (api *Router) deleteAlbumImage() http.HandlerFunc {
	return handleImageDelete(func(ctx context.Context) error {
		albumID := chi.URLParamFromCtx(ctx, "id")
		al, err := api.ds.Album(ctx).Get(albumID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return model.ErrNotFound
			}
			return err
		}
		if err := api.imgUpload.RemoveImage(ctx, al.UploadedImagePath()); err != nil {
			return err
		}
		return api.ds.Album(ctx).UpdateImage(al.ID, "")
	})
}
