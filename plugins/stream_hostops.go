package plugins

import (
	"errors"
	"io"
	"math"
)

const maxStreamChunk = 1 << 20

var errBadBuffer = errors.New("invalid stream buffer")

// guestMemory is the part of Extism memory the stream host functions use, so they can be tested without WASM.
type guestMemory interface {
	blockLength(offset uint64) uint64
	view(offset uint64, n uint32) ([]byte, bool)
	writeBlock(data []byte) (uint64, error)
}

func handleFromWire(v int64) (uint32, bool) {
	if v < 1 || v > math.MaxInt32 {
		return 0, false
	}
	return uint32(v), true
}

func bufferView(mem guestMemory, buf uint64, n int64) ([]byte, error) {
	if n < 1 || n > math.MaxUint32 {
		return nil, errBadBuffer
	}
	if uint64(n) > mem.blockLength(buf) {
		return nil, errBadBuffer
	}
	view, ok := mem.view(buf, uint32(n))
	if !ok {
		return nil, errBadBuffer
	}
	return view, nil
}

func (s *callScope) streamFor(handle int64, need streamRights) (*streamBackend, error) {
	h, ok := handleFromWire(handle)
	if !ok {
		return nil, errUnknownStream
	}
	return s.lookup(h, need)
}

func (s *callScope) fail(err error) int64 {
	s.lastErr = err
	return -1
}

func (s *callScope) hostRead(mem guestMemory, handle int64, buf uint64, n int64) int64 {
	b, err := s.streamFor(handle, streamRead)
	if err != nil {
		return s.fail(err)
	}
	view, err := bufferView(mem, buf, n)
	if err != nil {
		return s.fail(err)
	}
	if len(view) > maxStreamChunk {
		view = view[:maxStreamChunk]
	}
	got, err := b.Read(view)
	switch {
	case got > 0:
		s.lastErr = nil
		return int64(got)
	case errors.Is(err, io.EOF):
		s.lastErr = nil
		return 0
	default:
		return s.fail(err)
	}
}

func (s *callScope) hostWrite(mem guestMemory, handle int64, buf uint64, n int64) int64 {
	b, err := s.streamFor(handle, streamWrite)
	if err != nil {
		return s.fail(err)
	}
	view, err := bufferView(mem, buf, n)
	if err != nil {
		return s.fail(err)
	}
	written, err := b.Write(view)
	if written > 0 {
		s.lastErr = err
		return int64(written)
	}
	return s.fail(err)
}

func (s *callScope) hostCopy(dstHandle, srcHandle int64) int64 {
	dst, err := s.streamFor(dstHandle, streamWrite)
	if err != nil {
		s.lastErr = err
		return 0
	}
	src, err := s.streamFor(srcHandle, streamRead)
	if err != nil {
		s.lastErr = err
		return 0
	}
	n, err := io.Copy(dst, src)
	s.lastErr = err
	return n
}

func (s *callScope) hostClose(handle int64) int64 {
	h, ok := handleFromWire(handle)
	if !ok {
		return s.fail(errUnknownStream)
	}
	b, err := s.detach(h)
	if err != nil {
		return s.fail(err)
	}
	if err := b.Close(); err != nil {
		return s.fail(err)
	}
	s.lastErr = nil
	return 0
}

// hostError returns -1 when the message cannot be allocated (Extism signals that with offset 0, not an error),
// so the plugin never mistakes a failure for success.
func (s *callScope) hostError(mem guestMemory) int64 {
	if s.lastErr == nil {
		return 0
	}
	msg := s.lastErr.Error()
	if msg == "" {
		msg = "stream operation failed"
	}
	off, err := mem.writeBlock([]byte(msg))
	if err != nil || off == 0 || off > math.MaxInt64 {
		return -1
	}
	return int64(off)
}
