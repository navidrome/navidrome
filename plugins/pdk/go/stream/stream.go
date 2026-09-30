//go:build wasip1

// Package stream gives plugins io.Reader and io.Writer access to byte streams owned by Navidrome.
// Use one Stream per handle; Close frees its buffer.
package stream

import (
	"errors"
	"io"

	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
)

//go:wasmimport extism:host/user stream_read
func streamRead(handle int64, buf uint64, n int64) int64

//go:wasmimport extism:host/user stream_write
func streamWrite(handle int64, buf uint64, n int64) int64

//go:wasmimport extism:host/user stream_copy
func streamCopy(dst, src int64) int64

//go:wasmimport extism:host/user stream_close
func streamClose(handle int64) int64

//go:wasmimport extism:host/user stream_error
func streamError() int64

const bufferSize = 64 * 1024

var errStreamFailed = errors.New("stream operation failed")

// Stream is a handle to a byte stream owned by Navidrome. It reuses one memory block for all reads and writes.
type Stream struct {
	handle uint32
	buf    *pdk.Memory
}

func New(handle uint32) *Stream { return &Stream{handle: handle} }

func (s *Stream) Handle() uint32 { return s.handle }

func (s *Stream) buffer() *pdk.Memory {
	if s.buf == nil {
		m := pdk.Allocate(bufferSize)
		s.buf = &m
	}
	return s.buf
}

func (s *Stream) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	buf := s.buffer()
	r := streamRead(int64(s.handle), buf.Offset(), int64(min(len(p), bufferSize)))
	switch {
	case r > 0:
		buf.Load(p[:r])
		return int(r), nil
	case r == 0:
		return 0, io.EOF
	default:
		return 0, lastError()
	}
}

func (s *Stream) Write(p []byte) (int, error) {
	buf := s.buffer()
	written := 0
	for written < len(p) {
		chunk := p[written:min(len(p), written+bufferSize)]
		buf.Store(chunk)
		r := streamWrite(int64(s.handle), buf.Offset(), int64(len(chunk)))
		if r < 0 {
			return written, lastError()
		}
		written += int(r)
		if int(r) < len(chunk) {
			return written, lastError()
		}
	}
	return written, nil
}

func (s *Stream) Close() error {
	if s.buf != nil {
		s.buf.Free()
		s.buf = nil
	}
	if streamClose(int64(s.handle)) < 0 {
		return lastError()
	}
	return nil
}

// Copy moves all bytes from src to dst inside Navidrome; they never enter plugin memory.
func Copy(dst, src *Stream) (int64, error) {
	n := streamCopy(int64(dst.handle), int64(src.handle))
	if err := pendingError(); err != nil {
		return max(n, 0), err
	}
	if n < 0 {
		return 0, errStreamFailed
	}
	return n, nil
}

func lastError() error {
	if err := pendingError(); err != nil {
		return err
	}
	return errStreamFailed
}

func pendingError() error {
	ptr := streamError()
	switch {
	case ptr == 0:
		return nil
	case ptr < 0:
		return errStreamFailed
	}
	mem := pdk.FindMemory(uint64(ptr))
	defer mem.Free()
	return errors.New(string(mem.ReadBytes()))
}
