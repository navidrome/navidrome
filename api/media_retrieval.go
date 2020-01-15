package api

import (
	"io"
	"net/http"
	"os"

	"github.com/cloudsonic/sonic-server/api/responses"
	"github.com/cloudsonic/sonic-server/engine"
	"github.com/cloudsonic/sonic-server/log"
	"github.com/cloudsonic/sonic-server/model"
)

type MediaRetrievalController struct {
	cover engine.Cover
}

func NewMediaRetrievalController(cover engine.Cover) *MediaRetrievalController {
	return &MediaRetrievalController{cover: cover}
}

func (c *MediaRetrievalController) GetAvatar(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	var f *os.File
	f, err := os.Open("static/itunes.png")
	if err != nil {
		log.Error(r, "Image not found", err)
		return nil, NewError(responses.ErrorDataNotFound, "Avatar image not found")
	}
	defer f.Close()
	io.Copy(w, f)

	return nil, nil
}

func (c *MediaRetrievalController) GetCoverArt(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	id, err := RequiredParamString(r, "id", "id parameter required")
	if err != nil {
		return nil, err
	}
	size := ParamInt(r, "size", 0)

	err = c.cover.Get(id, size, w)

	switch {
	case err == model.ErrNotFound:
		log.Error(r, err.Error(), "id", id)
		return nil, NewError(responses.ErrorDataNotFound, "Cover not found")
	case err != nil:
		log.Error(r, err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	return nil, nil
}
