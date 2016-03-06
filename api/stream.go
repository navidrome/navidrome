package api

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/stream"
	"github.com/deluan/gosonic/utils"
	"github.com/karlkfi/inject"
	"strconv"
)

type StreamController struct {
	BaseAPIController
	repo domain.MediaFileRepository
	id   string
	mf   *domain.MediaFile
}

func (c *StreamController) Prepare() {
	inject.ExtractAssignable(utils.Graph, &c.repo)

	c.id = c.GetParameter("id", "id parameter required")

	mf, err := c.repo.Get(c.id)
	if err != nil {
		beego.Error("Error reading mediafile", c.id, "from the database", ":", err)
		c.SendError(responses.ERROR_GENERIC, "Internal error")
	}

	if mf == nil {
		beego.Error("MediaFile", c.id, "not found!")
		c.SendError(responses.ERROR_DATA_NOT_FOUND)
	}

	c.mf = mf
}

func (c *StreamController) Stream() {
	var maxBitRate int
	c.Ctx.Input.Bind(&maxBitRate, "maxBitRate")
	maxBitRate = utils.MinInt(c.mf.BitRate, maxBitRate)

	beego.Debug("Streaming file", ":", c.mf.Path)
	beego.Debug("Bitrate", c.mf.BitRate, "MaxBitRate", maxBitRate)

	if maxBitRate > 0 {
		c.Ctx.Output.Header("Content-Length", strconv.Itoa(c.mf.Duration*maxBitRate*1000/8))
	}
	c.Ctx.Output.Header("Content-Type", "audio/mpeg")
	c.Ctx.Output.Header("Expires", "0")
	c.Ctx.Output.Header("Cache-Control", "must-revalidate")
	c.Ctx.Output.Header("Pragma", "public")

	err := stream.Stream(c.mf.Path, c.mf.BitRate, maxBitRate, c.Ctx.ResponseWriter)
	if err != nil {
		beego.Error("Error streaming file id", c.id, ":", err)
	}

	beego.Debug("Finished streaming of", c.mf.Path)
}

func (c *StreamController) Download() {
	beego.Debug("Sending file", c.mf.Path)

	stream.Stream(c.mf.Path, 0, 0, c.Ctx.ResponseWriter)

	beego.Debug("Finished sending", c.mf.Path)
}
