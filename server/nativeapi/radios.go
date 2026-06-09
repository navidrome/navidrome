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
