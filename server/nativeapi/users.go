package nativeapi

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server"
)

func (api *Router) addUserRoute(r chi.Router) {
	constructor := func(ctx context.Context) rest.Repository {
		return api.users.NewRepository(ctx)
	}
	r.Route("/user", func(r chi.Router) {
		r.Get("/", rest.GetAll(constructor))
		r.Post("/", rest.Post(constructor))
		r.Route("/{id}", func(r chi.Router) {
			r.Use(server.URLParamsMiddleware)
			r.Get("/", rest.Get(constructor))
			r.Put("/", rest.Put(constructor))
			r.Delete("/", rest.Delete(constructor))
			r.Post("/image", api.uploadUserAvatar())
			r.Delete("/image", api.deleteUserAvatar())
		})
	})
}

// canEditAvatar allows the target user or any admin, and honors the feature flag for everyone.
func canEditAvatar(ctx context.Context, targetID string) error {
	if !conf.Server.EnableUserAvatarUpload {
		return model.ErrNotAuthorized
	}
	usr, _ := request.UserFrom(ctx)
	if !usr.IsAdmin && usr.ID != targetID {
		return model.ErrNotAuthorized
	}
	return nil
}

// checkAvatarPermission gates avatar uploads by EnableUserAvatarUpload, never by EnableArtworkUpload.
func checkAvatarPermission(w http.ResponseWriter, r *http.Request) bool {
	ctx := r.Context()
	if err := canEditAvatar(ctx, chi.URLParamFromCtx(ctx, "id")); err != nil {
		http.Error(w, "not authorized", http.StatusForbidden)
		return false
	}
	return true
}

func (api *Router) uploadUserAvatar() http.HandlerFunc {
	return handleImageUploadGated(checkAvatarPermission, func(ctx context.Context, reader io.Reader, ext string) error {
		userID := chi.URLParamFromCtx(ctx, "id")
		usr, err := api.ds.User(ctx).Get(userID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return model.ErrNotFound
			}
			return err
		}
		filename, err := api.imgUpload.SetAvatar(ctx, usr.ID, usr.UserName, usr.UploadedImagePath(), reader, ext)
		if err != nil {
			return err
		}
		return api.ds.User(ctx).UpdateImage(usr.ID, filename)
	})
}

func (api *Router) deleteUserAvatar() http.HandlerFunc {
	return handleImageDeleteGated(checkAvatarPermission, func(ctx context.Context) error {
		userID := chi.URLParamFromCtx(ctx, "id")
		usr, err := api.ds.User(ctx).Get(userID)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return model.ErrNotFound
			}
			return err
		}
		if err := api.imgUpload.RemoveImage(ctx, usr.UploadedImagePath()); err != nil {
			return err
		}
		return api.ds.User(ctx).UpdateImage(usr.ID, "")
	})
}
