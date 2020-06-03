package subsonic

import (
	"io"
	"net/http"

	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/engine"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/resources"
	"github.com/deluan/navidrome/server/subsonic/responses"
	"github.com/deluan/navidrome/utils"
)

type MediaRetrievalController struct {
	cover engine.Cover
}

func NewMediaRetrievalController(cover engine.Cover) *MediaRetrievalController {
	return &MediaRetrievalController{cover: cover}
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

	w.Header().Set("cache-control", "public, max-age=300")
	err = c.cover.Get(r.Context(), id, size, w)

	switch {
	case err == model.ErrNotFound:
		log.Error(r, "Couldn't find coverArt", "id", id, err)
		return nil, NewError(responses.ErrorDataNotFound, "Cover not found")
	case err != nil:
		log.Error(r, "Error retrieving coverArt", "id", id, err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	return nil, nil
}
