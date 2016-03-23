package api

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/engine"
	"github.com/deluan/gosonic/utils"
)

type StreamController struct {
	BaseAPIController
	repo domain.MediaFileRepository
	id   string
	mf   *domain.MediaFile
}

func (c *StreamController) Prepare() {
	utils.ResolveDependencies(&c.repo)

	c.id = c.RequiredParamString("id", "id parameter required")

	mf, err := c.repo.Get(c.id)
	switch {
	case err == domain.ErrNotFound:
		beego.Error("MediaFile", c.id, "not found!")
		c.SendError(responses.ErrorDataNotFound)
	case err != nil:
		beego.Error("Error reading mediafile", c.id, "from the database", ":", err)
		c.SendError(responses.ErrorGeneric, "Internal error")
	}

	c.mf = mf
}

// TODO Still getting the "Conn.Write wrote more than the declared Content-Length" error.
// Don't know if this causes any issues
func (c *StreamController) Stream() {
	maxBitRate := c.ParamInt("maxBitRate", 0)
	maxBitRate = utils.MinInt(c.mf.BitRate, maxBitRate)

	beego.Debug("Streaming file", c.id, ":", c.mf.Path)
	beego.Debug("Bitrate", c.mf.BitRate, "MaxBitRate", maxBitRate)

	// TODO Send proper estimated content-length
	//contentLength := c.mf.Size
	//if maxBitRate > 0 {
	//	contentLength = strconv.Itoa((c.mf.Duration + 1) * maxBitRate * 1000 / 8)
	//}
	c.Ctx.Output.Header("Content-Length", c.mf.Size)
	c.Ctx.Output.Header("Content-Type", "audio/mpeg")
	c.Ctx.Output.Header("Expires", "0")
	c.Ctx.Output.Header("Cache-Control", "must-revalidate")
	c.Ctx.Output.Header("Pragma", "public")

	if c.Ctx.Request.Method == "HEAD" {
		beego.Debug("Just a HEAD. Not streaming", c.mf.Path)
		return
	}

	err := engine.Stream(c.mf.Path, c.mf.BitRate, maxBitRate, c.Ctx.ResponseWriter)
	if err != nil {
		beego.Error("Error streaming file", c.id, ":", err)
	}

	beego.Debug("Finished streaming of", c.mf.Path)
}

func (c *StreamController) Download() {
	beego.Debug("Sending file", c.mf.Path)

	engine.Stream(c.mf.Path, 0, 0, c.Ctx.ResponseWriter)

	beego.Debug("Finished sending", c.mf.Path)
}
