package subsonic

import (
	"net/http"

	"github.com/deluan/navidrome/engine"
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

	fs, err := c.streamer.NewFileSystem(r.Context(), maxBitRate, format)
	if err != nil {
		return nil, err
	}

	// To be able to use a http.FileSystem, we need to change the URL structure
	r.URL.Path = id

	http.FileServer(fs).ServeHTTP(w, r)
	return nil, nil
}

func (c *StreamController) Download(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	id, err := RequiredParamString(r, "id", "id parameter required")
	if err != nil {
		return nil, err
	}

	fs, err := c.streamer.NewFileSystem(r.Context(), 0, "raw")
	if err != nil {
		return nil, err
	}

	// To be able to use a http.FileSystem, we need to change the URL structure
	r.URL.Path = id

	http.FileServer(fs).ServeHTTP(w, r)
	return nil, nil
}
