package subsonic

import (
	"io"
	"net/http"
	"strconv"

	"github.com/deluan/navidrome/engine"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/server/subsonic/responses"
	"github.com/deluan/navidrome/utils"
)

type StreamController struct {
	streamer engine.MediaStreamer
}

func NewStreamController(streamer engine.MediaStreamer) *StreamController {
	return &StreamController{streamer: streamer}
}

func (c *StreamController) Stream(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	id, err := RequiredParamString(r, "id", "id parameter required")
	if err != nil {
		return nil, err
	}
	maxBitRate := utils.ParamInt(r, "maxBitRate", 0)
	format := utils.ParamString(r, "format")

	stream, err := c.streamer.NewStream(r.Context(), id, maxBitRate, format)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := stream.Close(); err != nil {
			log.Error("Error closing stream", "id", id, "file", stream.Name(), err)
		}
	}()

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Content-Duration", strconv.FormatFloat(float64(stream.Duration()), 'G', -1, 32))

	if stream.Seekable() {
		http.ServeContent(w, r, stream.Name(), stream.ModTime(), stream)
	} else {
		// If the stream doesn't provide a size (i.e. is not seekable), we can't support ranges/content-length
		w.Header().Set("Accept-Ranges", "none")
		w.Header().Set("Content-Type", stream.ContentType())
		if c, err := io.Copy(w, stream); err != nil {
			log.Error(r.Context(), "Error sending transcoded file", "id", id, err)
		} else {
			log.Trace(r.Context(), "Success sending transcode file", "id", id, "size", c)
		}
	}

	return nil, nil
}

func (c *StreamController) Download(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	id, err := RequiredParamString(r, "id", "id parameter required")
	if err != nil {
		return nil, err
	}

	stream, err := c.streamer.NewStream(r.Context(), id, 0, "raw")
	if err != nil {
		return nil, err
	}

	http.ServeContent(w, r, stream.Name(), stream.ModTime(), stream)
	return nil, nil
}
