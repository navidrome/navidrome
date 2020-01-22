package subsonic

import (
	"net/http"
	"strconv"

	"github.com/cloudsonic/sonic-server/engine"
	"github.com/cloudsonic/sonic-server/log"
	"github.com/cloudsonic/sonic-server/model"
	"github.com/cloudsonic/sonic-server/server/subsonic/responses"
	"github.com/cloudsonic/sonic-server/utils"
)

type StreamController struct {
	browser engine.Browser
}

func NewStreamController(browser engine.Browser) *StreamController {
	return &StreamController{browser: browser}
}

func (c *StreamController) getMediaFile(r *http.Request) (mf *engine.Entry, err error) {
	id, err := RequiredParamString(r, "id", "id parameter required")
	if err != nil {
		return nil, err
	}

	mf, err = c.browser.GetSong(r.Context(), id)
	switch {
	case err == model.ErrNotFound:
		log.Error(r, "Mediafile not found", "id", id)
		return nil, NewError(responses.ErrorDataNotFound)
	case err != nil:
		log.Error(r, "Error reading mediafile from DB", "id", id, err)
		return nil, NewError(responses.ErrorGeneric, "Internal error")
	}
	return
}

// TODO Still getting the "Conn.Write wrote more than the declared Content-Length" error.
// Don't know if this causes any issues
func (c *StreamController) Stream(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	mf, err := c.getMediaFile(r)
	if err != nil {
		return nil, err
	}
	maxBitRate := ParamInt(r, "maxBitRate", 0)
	maxBitRate = utils.MinInt(mf.BitRate, maxBitRate)

	log.Debug(r, "Streaming file", "id", mf.Id, "path", mf.AbsolutePath, "bitrate", mf.BitRate, "maxBitRate", maxBitRate)

	// TODO Send proper estimated content-length
	//contentLength := mf.Size
	//if maxBitRate > 0 {
	//	contentLength = strconv.Itoa((mf.Duration + 1) * maxBitRate * 1000 / 8)
	//}
	h := w.Header()
	h.Set("Content-Length", strconv.Itoa(mf.Size))
	h.Set("Content-Type", "audio/mpeg")
	h.Set("Expires", "0")
	h.Set("Cache-Control", "must-revalidate")
	h.Set("Pragma", "public")

	if r.Method == "HEAD" {
		log.Debug(r, "Just a HEAD. Not streaming", "path", mf.AbsolutePath)
		return nil, nil
	}

	err = engine.Stream(r.Context(), mf.AbsolutePath, mf.BitRate, maxBitRate, w)
	if err != nil {
		log.Error(r, "Error streaming file", "id", mf.Id, err)
	}

	log.Debug(r, "Finished streaming", "path", mf.AbsolutePath)
	return nil, nil
}

func (c *StreamController) Download(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	mf, err := c.getMediaFile(r)
	if err != nil {
		return nil, err
	}
	log.Debug(r, "Sending file", "path", mf.AbsolutePath)

	err = engine.Stream(r.Context(), mf.AbsolutePath, 0, 0, w)
	if err != nil {
		log.Error(r, "Error downloading file", "path", mf.AbsolutePath, err)
	}

	log.Debug(r, "Finished sending", "path", mf.AbsolutePath)

	return nil, nil
}
