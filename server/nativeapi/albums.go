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
		oldPath, err := api.albumImagePathToRemove(ctx, al)
		if err != nil {
			return err
		}
		filename, err := api.imgUpload.SetImage(ctx, consts.EntityAlbum, al.ID, al.Name, oldPath, reader, ext)
		if err != nil {
			return err
		}
		return api.ds.Album(ctx).UpdateImage(al.ID, filename)
	})
}

// albumImagePathToRemove returns the album's current image path, or "" when the file is
// shared with another album row (post album-ID copy) and must be left on disk.
func (api *Router) albumImagePathToRemove(ctx context.Context, al *model.Album) (string, error) {
	path := al.UploadedImagePath()
	if path == "" {
		return "", nil
	}
	refs, err := api.ds.Album(ctx).CountByImage(al.UploadedImage)
	if err != nil {
		return "", err
	}
	if refs > 1 {
		return "", nil
	}
	return path, nil
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
		oldPath, err := api.albumImagePathToRemove(ctx, al)
		if err != nil {
			return err
		}
		if err := api.imgUpload.RemoveImage(ctx, oldPath); err != nil {
			return err
		}
		return api.ds.Album(ctx).UpdateImage(al.ID, "")
	})
}
