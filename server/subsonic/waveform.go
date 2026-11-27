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
	if !conf.Server().EnableWaveform || api.waveform == nil {
		return nil, newError(responses.ErrorGeneric, "waveforms are disabled on this server")
	}
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}
	// Without samples, all stored levels are returned.
	samples := 0
	if p.StringOr("samples", "") != "" {
		samples, err = p.Int("samples")
		if err != nil {
			return nil, err
		}
		if samples <= 0 {
			return nil, newError(responses.ErrorGeneric, "samples must be a positive number")
		}
	}

	mf, err := api.ds.MediaFile().Get(r.Context(), id)
	if err != nil {
		return nil, err
	}
	levels, err := api.waveform.Get(r.Context(), mf)
	if errors.Is(err, context.Canceled) {
		return nil, err
	}
	if err != nil {
		// Generation errors can carry file paths and ffmpeg output, so they are only logged.
		log.Error(r, "Error generating waveform", "id", mf.ID, "path", mf.Path, err)
		return nil, newError(responses.ErrorGeneric, "could not generate waveform")
	}

	out := waveform.Downsample(levels, samples)
	interval := float64(waveform.IntervalMs)
	if len(out) > 0 {
		interval = float64(waveform.IntervalMs*len(levels)) / float64(len(out))
	}
	wf := &responses.Waveform{
		ID:       mf.ID,
		Duration: mf.Duration,
		Interval: interval,
		Samples:  int32(len(out)),
		Levels:   make([]float64, len(out)),
	}
	for i, v := range out {
		wf.Levels[i] = float64(v) / 100
	}
	response := newResponse()
	response.Waveform = wf
	return response, nil
}
