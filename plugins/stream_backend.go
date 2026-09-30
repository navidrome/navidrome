package plugins

import (
	"errors"
	"io"
	"sync"
)

type streamRights uint8

const (
	streamRead streamRights = 1 << iota
	streamWrite
)

// maxEmptyReads bounds how often a Read may return (0, nil); io.Copy would otherwise loop forever.
const maxEmptyReads = 100

var (
	errStreamNotReadable = errors.New("stream is not readable")
	errStreamNotWritable = errors.New("stream is not writable")
)

// streamBackend is what a handle points to. It keeps an error that arrived together with data, so every later
// consumer (plugin reads, stream_copy, host code after transfer) still sees it.
type streamBackend struct {
	value  any
	r      io.Reader
	w      io.Writer
	c      io.Closer
	rights streamRights

	mu       sync.Mutex
	retained error

	closeOnce sync.Once
	closeErr  error
}

func newReadStream(rc io.ReadCloser) *streamBackend {
	return &streamBackend{value: rc, r: rc, c: rc, rights: streamRead}
}

func newWriteStream(wc io.WriteCloser) *streamBackend {
	return &streamBackend{value: wc, w: wc, c: wc, rights: streamWrite}
}

func (b *streamBackend) Read(p []byte) (int, error) {
	if err := b.retainedErr(); err != nil {
		return 0, err
	}
	if b.r == nil {
		return 0, errStreamNotReadable
	}
	if len(p) == 0 {
		return 0, nil
	}
	for range maxEmptyReads {
		n, err := b.r.Read(p)
		if err != nil {
			b.retain(err)
			if n > 0 {
				return n, nil
			}
			return 0, err
		}
		if n > 0 {
			return n, nil
		}
	}
	b.retain(io.ErrNoProgress)
	return 0, io.ErrNoProgress
}

func (b *streamBackend) Write(p []byte) (int, error) {
	if err := b.retainedErr(); err != nil {
		return 0, err
	}
	if b.w == nil {
		return 0, errStreamNotWritable
	}
	n, err := b.w.Write(p)
	if err == nil && n < len(p) {
		err = io.ErrShortWrite
	}
	if err != nil {
		b.retain(err)
	}
	return n, err
}

func (b *streamBackend) Close() error {
	b.closeOnce.Do(func() { b.closeErr = b.c.Close() })
	return b.closeErr
}

// transferValue is what host code gets when a plugin returns this stream: the original value keeps abilities
// like io.Seeker, unless a retained error would be lost with it.
func (b *streamBackend) transferValue() any {
	if b.retainedErr() != nil {
		return b
	}
	return b.value
}

func (b *streamBackend) retain(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.retained == nil {
		b.retained = err
	}
}

func (b *streamBackend) retainedErr() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.retained
}
