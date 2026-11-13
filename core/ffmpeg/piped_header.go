package ffmpeg

import (
	"bytes"
	"errors"
	"io"
)

// patchPipedHeader repairs the header ffmpeg cannot finish when its output is a pipe,
// since it only learns the sample or frame count once it can no longer rewind.
func patchPipedHeader(r io.ReadCloser, duration float32) io.ReadCloser {
	if duration <= 0 {
		return r
	}
	return &headerPatcher{ReadCloser: r, duration: duration}
}

type headerPatcher struct {
	io.ReadCloser
	duration float32
	// Peeking here rather than in the constructor keeps Transcode from blocking
	// until ffmpeg has emitted its first bytes.
	stream io.Reader
}

func (h *headerPatcher) Read(p []byte) (int, error) {
	if h.stream == nil {
		prefix, err := h.peek()
		if err != nil {
			return 0, err
		}
		h.stream = io.MultiReader(bytes.NewReader(prefix), h.ReadCloser)
	}
	return h.stream.Read(p)
}

// peek dispatches on the magic bytes rather than on the requested format, which is a
// free-text label on the transcoding profile and need not match what ffmpeg emits.
func (h *headerPatcher) peek() ([]byte, error) {
	buf, err := h.fill(nil, 4)
	if err != nil || len(buf) < 4 {
		return buf, err
	}
	if string(buf) == "fLaC" {
		return h.flacPrefix(buf)
	}
	return h.mp3Prefix(buf)
}

// fill grows buf to n bytes, stopping short when the stream ends first.
func (h *headerPatcher) fill(buf []byte, n int) ([]byte, error) {
	if len(buf) >= n {
		return buf, nil
	}
	more := make([]byte, n-len(buf))
	read, err := io.ReadFull(h.ReadCloser, more)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return buf, err
	}
	return append(buf, more[:read]...), nil
}
