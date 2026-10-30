package subsonic

import (
	"net/http"

	"github.com/navidrome/navidrome/core/waveform"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
)

func (api *Router) GetWaveform(r *http.Request) (*responses.Subsonic, error) {
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
	if err != nil {
		return nil, err
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
