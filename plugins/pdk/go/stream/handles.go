package stream

import (
	"errors"
	"io"
)

var errNotHostStream = errors.New("only streams from Navidrome can be passed to a host function")

// Reader returns the stream a host function returned, or nil for handle 0 (no stream).
func Reader(h uint32) io.ReadCloser {
	if h == 0 {
		return nil
	}
	return New(h)
}

// Writer returns the stream a host function returned, or nil for handle 0 (no stream).
func Writer(h uint32) io.WriteCloser {
	if h == 0 {
		return nil
	}
	return New(h)
}

// HandleOf returns the handle to pass to a host function, which takes the stream over. Streams a plugin made
// itself cannot cross.
func HandleOf(v any) (uint32, error) {
	switch s := v.(type) {
	case nil:
		return 0, nil
	case *Stream:
		if s == nil {
			return 0, nil
		}
		s.releaseBuffer()
		return s.handle, nil
	default:
		return 0, errNotHostStream
	}
}
