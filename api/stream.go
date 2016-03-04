package api

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/stream"
	"github.com/deluan/gosonic/utils"
	"github.com/karlkfi/inject"
	"io"
	"net/http"
	"strconv"
)

type StreamController struct {
	BaseAPIController
	repo domain.MediaFileRepository
	id   string
	mf   *domain.MediaFile
}

type flushWriter struct {
	f http.Flusher
	w io.Writer
}

func (fw *flushWriter) Write(p []byte) (n int, err error) {
	n, err = fw.w.Write(p)
	if fw.f != nil {
		fw.f.Flush()
	}
	return
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

func createFlusher(w http.ResponseWriter) io.Writer {
	fw := flushWriter{w: w}
	if f, ok := w.(http.Flusher); ok {
		fw.f = f
	}
	return &fw
}

// TODO Investigate why it is not flushing before closing the connection
func (c *StreamController) Stream() {
	var maxBitRate int
	c.Ctx.Input.Bind(&maxBitRate, "maxBitRate")
	maxBitRate = utils.MinInt(c.mf.BitRate, maxBitRate)

	beego.Debug("Streaming file", maxBitRate, ":", c.mf.Path)
	beego.Debug("Bitrate", c.mf.BitRate, "MaxBitRate", maxBitRate)

	if maxBitRate > 0 {
		c.Ctx.Output.Header("Content-Length", strconv.Itoa(c.mf.Duration*maxBitRate*1000/8))
	}
	c.Ctx.Output.Header("Content-Type", "audio/mpeg")
	c.Ctx.Output.Header("Expires", "0")
	c.Ctx.Output.Header("Cache-Control", "must-revalidate")
	c.Ctx.Output.Header("Pragma", "public")

	err := stream.Stream(c.mf.Path, c.mf.BitRate, maxBitRate, createFlusher(c.Ctx.ResponseWriter))
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
