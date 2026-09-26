package ffmpeg

import (
	"encoding/binary"
	"math"
)

const (
	flacPrefixLen       = 26 // through the last total_samples byte
	flacMaxTotalSamples = 1<<36 - 1
)

// flacPrefix fills in the STREAMINFO total_samples that ffmpeg leaves at 0 when
// writing to a pipe, since a decoder cannot seek a cached FLAC without it.
func (h *headerPatcher) flacPrefix(buf []byte) ([]byte, error) {
	buf, err := h.fill(buf, flacPrefixLen)
	if err != nil || len(buf) < flacPrefixLen {
		return buf, err
	}
	setFLACTotalSamples(buf, h.duration)
	return buf, nil
}

// setFLACTotalSamples takes the rate from the header rather than the transcode
// options, so a resampled (-ar) output still gets the right count.
func setFLACTotalSamples(prefix []byte, duration float32) {
	if prefix[4]&0x7F != 0 { // the first metadata block must be STREAMINFO
		return
	}
	// 20-bit rate | 3-bit channels | 5-bit depth | 36-bit total_samples
	info := binary.BigEndian.Uint64(prefix[18:])
	rate := info >> 44
	if rate == 0 || info&flacMaxTotalSamples != 0 {
		return
	}
	total := math.Round(float64(duration) * float64(rate))
	if total > flacMaxTotalSamples {
		return
	}
	binary.BigEndian.PutUint64(prefix[18:], info|uint64(total))
}
