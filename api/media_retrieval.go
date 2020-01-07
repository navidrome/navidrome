package api

import (
	"io"
	"net/http"
	"os"

	"github.com/astaxie/beego"
	"github.com/cloudsonic/sonic-server/api/responses"
	"github.com/cloudsonic/sonic-server/domain"
	"github.com/cloudsonic/sonic-server/engine"
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
		beego.Error(err, "Image not found")
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
	case err == domain.ErrNotFound:
		beego.Error(err, "Id:", id)
		return nil, NewError(responses.ErrorDataNotFound, "Cover not found")
	case err != nil:
		beego.Error(err)
		return nil, NewError(responses.ErrorGeneric, "Internal Error")
	}

	return nil, nil
}
