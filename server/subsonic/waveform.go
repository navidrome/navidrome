package subsonic

import (
	"context"
	"errors"
	"net/http"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/waveform"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
)

// GetWaveform is always routed, so a client calling it on a server with waveforms turned off gets a
// regular Subsonic error instead of a non-Subsonic 501 page. Clients should check for the
// songWaveform extension before calling it.
func (api *Router) GetWaveform(r *http.Request) (*responses.Subsonic, error) {
	if !conf.Server.EnableWaveform || api.waveform == nil {
		return nil, newError(responses.ErrorGeneric, "waveforms are disabled on this server")
	}
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}
	samples := p.IntOr("samples", waveform.DefaultSamples)
	if samples <= 0 {
		return nil, newError(responses.ErrorGeneric, "samples must be a positive number")
	}
	samples = min(samples, waveform.MaxSamples)

	mf, err := api.ds.MediaFile().Get(r.Context(), id)
	if err != nil {
		return nil, err
	}
	peaks, err := api.waveform.Get(r.Context(), mf, samples)
	if errors.Is(err, context.Canceled) {
		return nil, err
	}
	if err != nil {
		// Generation errors can carry file paths and ffmpeg output, so they are only logged.
		log.Error(r, "Error generating waveform", "id", mf.ID, "path", mf.Path, err)
		return nil, newError(responses.ErrorGeneric, "could not generate waveform")
	}

	wf := &responses.Waveform{
		ID:       mf.ID,
		Duration: mf.Duration,
		Samples:  int32(len(peaks)),
		Peaks:    make([]int32, len(peaks)),
	}
	for i, v := range peaks {
		wf.Peaks[i] = int32(v)
	}
	response := newResponse()
	response.Waveform = wf
	return response, nil
}
