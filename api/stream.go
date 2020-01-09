package api

import (
	"net/http"

	"github.com/cloudsonic/sonic-server/api/responses"
	"github.com/cloudsonic/sonic-server/domain"
	"github.com/cloudsonic/sonic-server/engine"
	"github.com/cloudsonic/sonic-server/log"
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
		log.Error(r, "Mediafile not found", "id", c.id)
		return NewError(responses.ErrorDataNotFound)
	case err != nil:
		log.Error(r, "Error reading mediafile from DB", "id", c.id, err)
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

	log.Debug(r, "Streaming file", "id", c.id, "path", c.mf.Path, "bitrate", c.mf.BitRate, "maxBitRate", maxBitRate)

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
		log.Debug(r, "Just a HEAD. Not streaming", "path", c.mf.Path)
		return nil, nil
	}

	err = engine.Stream(r.Context(), c.mf.Path, c.mf.BitRate, maxBitRate, w)
	if err != nil {
		log.Error(r, "Error streaming file", "id", c.id, err)
	}

	log.Debug(r, "Finished streaming", "path", c.mf.Path)
	return nil, nil
}

func (c *StreamController) Download(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	err := c.Prepare(r)
	if err != nil {
		return nil, err
	}
	log.Debug(r, "Sending file", "path", c.mf.Path)

	err = engine.Stream(r.Context(), c.mf.Path, 0, 0, w)
	if err != nil {
		log.Error(r, "Error downloading file", "path", c.mf.Path, err)
	}

	log.Debug(r, "Finished sending", "path", c.mf.Path)

	return nil, nil
}
