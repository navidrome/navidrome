package ffmpeg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
)

const (
	flacPrefixLen       = 26 // through the last total_samples byte
	flacMaxTotalSamples = 1<<36 - 1
)

// patchFLACDuration fills in the STREAMINFO total_samples that ffmpeg leaves at 0
// when writing to a pipe, since a decoder cannot seek a cached FLAC without it.
func patchFLACDuration(r io.ReadCloser, duration float32) io.ReadCloser {
	if duration <= 0 {
		return r
	}
	return &flacPatcher{ReadCloser: r, duration: duration}
}

type flacPatcher struct {
	io.ReadCloser
	duration float32
	// Peeking here rather than in the constructor keeps Transcode from blocking
	// until ffmpeg has emitted its first bytes.
	stream io.Reader
}

func (f *flacPatcher) Read(p []byte) (int, error) {
	if f.stream == nil {
		prefix := make([]byte, flacPrefixLen)
		n, err := io.ReadFull(f.ReadCloser, prefix)
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			return 0, err
		}
		prefix = prefix[:n]
		if err == nil {
			setFLACTotalSamples(prefix, f.duration)
		}
		f.stream = io.MultiReader(bytes.NewReader(prefix), f.ReadCloser)
	}
	return f.stream.Read(p)
}

// setFLACTotalSamples takes the rate from the header rather than the transcode
// options, so a resampled (-ar) output still gets the right count.
func setFLACTotalSamples(prefix []byte, duration float32) {
	if string(prefix[:4]) != "fLaC" || prefix[4]&0x7F != 0 {
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
