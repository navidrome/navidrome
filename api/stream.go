package api

import (
	"github.com/deluan/gosonic/utils"
	"github.com/karlkfi/inject"
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/api/responses"
	"github.com/astaxie/beego"
	"io"
	"os"
)


type StreamController struct {
	BaseAPIController
	repo domain.MediaFileRepository
}

func (c *StreamController) Prepare() {
	inject.ExtractAssignable(utils.Graph, &c.repo)
}

// For realtime transcoding, see : http://stackoverflow.com/questions/19292113/not-buffered-http-responsewritter-in-golang
func (c *StreamController) Get() {
	id := c.ValidateParameters("id", "id parameter required")

	mf, err := c.repo.Get(id)
	if err != nil {
		beego.Error("Error reading mediafile", id, "from the database", ":", err)
		c.SendError(responses.ERROR_GENERIC, "Internal error")
	}
	beego.Debug("Streaming file", mf.Path)

	f, err := os.Open(mf.Path)
	if err != nil {
		beego.Warn("Error opening file", mf.Path, "-", err)
		c.SendError(responses.ERROR_DATA_NOT_FOUND, "cover art not available")
	}

	c.Ctx.Output.ContentType(mf.ContentType())
	io.Copy(c.Ctx.ResponseWriter, f)

	beego.Debug("Finished streaming of", mf.Path)
}
