package subsonic

import (
	"io"
	"net/http"

	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/core"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/resources"
	"github.com/deluan/navidrome/server/subsonic/responses"
	"github.com/deluan/navidrome/utils"
)

type MediaRetrievalController struct {
	artwork core.Artwork
}

func NewMediaRetrievalController(artwork core.Artwork) *MediaRetrievalController {
	return &MediaRetrievalController{artwork: artwork}
}

func (c *MediaRetrievalController) GetAvatar(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	f, err := resources.AssetFile().Open(consts.PlaceholderAlbumArt)
	if err != nil {
		log.Error(r, "Image not found", err)
		return nil, NewError(responses.ErrorDataNotFound, "Avatar image not found")
	}
	defer f.Close()
	_, _ = io.Copy(w, f)

	return nil, nil
}

func (c *MediaRetrievalController) GetCoverArt(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	id, err := RequiredParamString(r, "id", "id parameter required")
	if err != nil {
		return nil, err
	}
	size := utils.ParamInt(r, "size", 0)

	w.Header().Set("cache-control", "public, max-age=315360000")
	err = c.artwork.Get(r.Context(), id, size, w)

	switch {
	case err == model.ErrNotFound:
		log.Error(r, "Couldn't find coverArt", "id", id, err)
		return nil, NewError(responses.ErrorDataNotFound, "Artwork not found")
	case err != nil:
		log.Error(r, "Error retrieving coverArt", "id", id, err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	return nil, nil
}
