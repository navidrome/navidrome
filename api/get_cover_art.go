package api

import (
	"github.com/deluan/gosonic/domain"
	"github.com/karlkfi/inject"
	"github.com/deluan/gosonic/utils"
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
	"io/ioutil"
"github.com/dhowden/tag"
	"os"
)

type GetCoverArtController struct {
	BaseAPIController
	repo domain.MediaFileRepository
}

func (c *GetCoverArtController) Prepare() {
	inject.ExtractAssignable(utils.Graph, &c.repo)
}

func (c *GetCoverArtController) Get() {
	id := c.Input().Get("id")
	if id == "" {
		c.SendError(responses.ERROR_MISSING_PARAMETER, "id parameter required")
	}

	mf, err := c.repo.Get(id)
	if err != nil {
		beego.Error("Error reading mediafile", id, "from the database", ":", err)
		c.SendError(responses.ERROR_GENERIC, "Internal error")
	}

	var img []byte

	if (mf.HasCoverArt) {
		img, err = readFromTag(mf.Path)
		beego.Debug("Serving cover art from", mf.Path)
	} else {
		img, err = ioutil.ReadFile("static/default_cover.jpg")
		beego.Debug("Serving default cover art")
	}

	if err != nil {
		beego.Error("Could not retrieve cover art", id, ":", err)
		c.SendError(responses.ERROR_DATA_NOT_FOUND, "cover art not available")
	}


	c.Ctx.Output.ContentType("image/jpg")
	c.Ctx.Output.Body(img)
}

func readFromTag(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		beego.Warn("Error opening file", path, "-", err)
		return nil, err
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil {
		beego.Warn("Error reading tag from file", path, "-", err)
		return nil, err
	}

	return m.Picture().Data, nil
}

