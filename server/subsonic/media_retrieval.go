package subsonic

import (
	"io"
	"net/http"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/core"
	"github.com/deluan/navidrome/core/gravatar"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/resources"
	"github.com/deluan/navidrome/server/subsonic/responses"
	"github.com/deluan/navidrome/utils"
)

type MediaRetrievalController struct {
	artwork core.Artwork
	ds      model.DataStore
}

func NewMediaRetrievalController(artwork core.Artwork, ds model.DataStore) *MediaRetrievalController {
	return &MediaRetrievalController{artwork: artwork, ds: ds}
}

func (c *MediaRetrievalController) GetAvatar(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	if !conf.Server.EnableGravatar {
		return c.getPlaceHolderAvatar(w, r)
	}
	username, err := requiredParamString(r, "username")
	if err != nil {
		return nil, err
	}
	ctx := r.Context()
	u, err := c.ds.User(ctx).FindByUsername(username)
	if err != nil {
		return nil, err
	}
	if u.Email == "" {
		log.Warn(ctx, "User needs an email for gravatar to work", "username", username)
		return c.getPlaceHolderAvatar(w, r)
	}
	http.Redirect(w, r, gravatar.Url(u.Email, 0), http.StatusFound)
	return nil, nil
}

func (c *MediaRetrievalController) getPlaceHolderAvatar(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	f, err := resources.AssetFile().Open(consts.PlaceholderAvatar)
	if err != nil {
		log.Error(r, "Image not found", err)
		return nil, newError(responses.ErrorDataNotFound, "Avatar image not found")
	}
	defer f.Close()
	_, _ = io.Copy(w, f)

	return nil, nil
}

func (c *MediaRetrievalController) GetCoverArt(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	id, err := requiredParamString(r, "id")
	if err != nil {
		return nil, err
	}
	size := utils.ParamInt(r, "size", 0)

	w.Header().Set("cache-control", "public, max-age=315360000")

	imgReader, err := c.artwork.Get(r.Context(), id, size)
	switch {
	case err == model.ErrNotFound:
		log.Error(r, "Couldn't find coverArt", "id", id, err)
		return nil, newError(responses.ErrorDataNotFound, "Artwork not found")
	case err != nil:
		log.Error(r, "Error retrieving coverArt", "id", id, err)
		return nil, err
	}

	defer imgReader.Close()
	_, err = io.Copy(w, imgReader)

	return nil, err
}
