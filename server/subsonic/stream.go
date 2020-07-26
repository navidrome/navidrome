package subsonic

import (
	"io"
	"net/http"
	"strconv"

	"github.com/deluan/navidrome/core"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/server/subsonic/responses"
	"github.com/deluan/navidrome/utils"
)

type StreamController struct {
	streamer core.MediaStreamer
}

func NewStreamController(streamer core.MediaStreamer) *StreamController {
	return &StreamController{streamer: streamer}
}

func (c *StreamController) Stream(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	id, err := RequiredParamString(r, "id", "id parameter required")
	if err != nil {
		return nil, err
	}
	maxBitRate := utils.ParamInt(r, "maxBitRate", 0)
	format := utils.ParamString(r, "format")
	estimateContentLength := utils.ParamBool(r, "estimateContentLength", false)

	stream, err := c.streamer.NewStream(r.Context(), id, format, maxBitRate)
	if err != nil {
		return nil, err
	}

	// Make sure the stream will be closed at the end, to avoid leakage
	defer func() {
		if err := stream.Close(); err != nil && log.CurrentLevel() >= log.LevelDebug {
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

		// if Client requests the estimated content-length, send it
		if estimateContentLength {
			length := strconv.Itoa(stream.EstimatedContentLength())
			log.Trace(r.Context(), "Estimated content-length", "contentLength", length)
			w.Header().Set("Content-Length", length)
		}

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

	stream, err := c.streamer.NewStream(r.Context(), id, "raw", 0)
	if err != nil {
		return nil, err
	}

	http.ServeContent(w, r, stream.Name(), stream.ModTime(), stream)
	return nil, nil
}
