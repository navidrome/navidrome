package api

import (
	"net/http"

	"github.com/astaxie/beego"
	"github.com/cloudsonic/sonic-server/api/responses"
	"github.com/cloudsonic/sonic-server/domain"
	"github.com/cloudsonic/sonic-server/engine"
	"github.com/cloudsonic/sonic-server/utils"
)

type StreamController struct {
	repo domain.MediaFileRepository
	id   string
	mf   *domain.MediaFile
}

func NewStreamController(repo domain.MediaFileRepository) *StreamController {
	return &StreamController{repo: repo}
}

func (c *StreamController) Prepare(r *http.Request) (err error) {
	c.id, err = RequiredParamString(r, "id", "id parameter required")
	if err != nil {
		return err
	}

	c.mf, err = c.repo.Get(c.id)
	switch {
	case err == domain.ErrNotFound:
		beego.Error("MediaFile", c.id, "not found!")
		return NewError(responses.ErrorDataNotFound)
	case err != nil:
		beego.Error("Error reading mediafile", c.id, "from the database", ":", err)
		return NewError(responses.ErrorGeneric, "Internal error")
	}
	return nil
}

// TODO Still getting the "Conn.Write wrote more than the declared Content-Length" error.
// Don't know if this causes any issues
func (c *StreamController) Stream(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	err := c.Prepare(r)
	if err != nil {
		return nil, err
	}
	maxBitRate := ParamInt(r, "maxBitRate", 0)
	maxBitRate = utils.MinInt(c.mf.BitRate, maxBitRate)

	beego.Debug("Streaming file", c.id, ":", c.mf.Path)
	beego.Debug("Bitrate", c.mf.BitRate, "MaxBitRate", maxBitRate)

	// TODO Send proper estimated content-length
	//contentLength := c.mf.Size
	//if maxBitRate > 0 {
	//	contentLength = strconv.Itoa((c.mf.Duration + 1) * maxBitRate * 1000 / 8)
	//}
	h := w.Header()
	h.Set("Content-Length", c.mf.Size)
	h.Set("Content-Type", "audio/mpeg")
	h.Set("Expires", "0")
	h.Set("Cache-Control", "must-revalidate")
	h.Set("Pragma", "public")

	if r.Method == "HEAD" {
		beego.Debug("Just a HEAD. Not streaming", c.mf.Path)
		return nil, nil
	}

	err = engine.Stream(c.mf.Path, c.mf.BitRate, maxBitRate, w)
	if err != nil {
		beego.Error("Error streaming file", c.id, ":", err)
	}

	beego.Debug("Finished streaming of", c.mf.Path)
	return nil, nil
}

func (c *StreamController) Download(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	err := c.Prepare(r)
	if err != nil {
		return nil, err
	}
	beego.Debug("Sending file", c.mf.Path)

	err = engine.Stream(c.mf.Path, 0, 0, w)
	if err != nil {
		beego.Error("Error downloading file", c.mf.Path, ":", err.Error())
	}

	beego.Debug("Finished sending", c.mf.Path)

	return nil, nil
}
