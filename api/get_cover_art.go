package api

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/utils"
	"github.com/dhowden/tag"
	"github.com/karlkfi/inject"
	"io/ioutil"
	"os"
)

type GetCoverArtController struct {
	BaseAPIController
	repo domain.MediaFileRepository
}

func (c *GetCoverArtController) Prepare() {
	inject.ExtractAssignable(utils.Graph, &c.repo)
}

// TODO accept size parameter
func (c *GetCoverArtController) Get() {
	id := c.GetParameter("id", "id parameter required")

	mf, err := c.repo.Get(id)
	if err != nil {
		beego.Error("Error reading mediafile", id, "from the database", ":", err)
		c.SendError(responses.ERROR_GENERIC, "Internal error")
	}

	var img []byte

	if mf != nil && mf.HasCoverArt {
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
