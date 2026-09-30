// Package streams lets generated host functions convert stream handles for the plugin call they run in.
package streams

import (
	"context"
	"errors"
	"io"
)

var errNoCall = errors.New("no plugin call in progress")

// Streams is the stream table of the plugin call a host function runs in.
type Streams interface {
	RegisterReader(io.ReadCloser) (uint32, error)
	RegisterWriter(io.WriteCloser) (uint32, error)
	TakeReader(handle uint32) (io.ReadCloser, error)
	TakeWriter(handle uint32) (io.WriteCloser, error)
	// Release closes a stream registered in this call whose handle never reached the plugin.
	Release(handle uint32)
}

type streamsKey struct{}

func WithStreams(ctx context.Context, s Streams) context.Context {
	return context.WithValue(ctx, streamsKey{}, s)
}

func From(ctx context.Context) (Streams, bool) {
	s, ok := ctx.Value(streamsKey{}).(Streams)
	return s, ok
}

// Conv converts the handles of one host function call. After the first error it converts nothing more and
// closes whatever it is handed, so no stream leaks when the call fails halfway.
type Conv struct {
	s          Streams
	taken      []io.Closer
	registered []uint32
	err        error
}

func NewConv(ctx context.Context) *Conv {
	s, ok := From(ctx)
	if !ok {
		return &Conv{err: errNoCall}
	}
	return &Conv{s: s}
}

func (c *Conv) Reader(h uint32) io.ReadCloser {
	if h == 0 || c.err != nil {
		return nil
	}
	rc, err := c.s.TakeReader(h)
	if err != nil {
		c.err = err
		return nil
	}
	c.taken = append(c.taken, rc)
	return rc
}

func (c *Conv) Writer(h uint32) io.WriteCloser {
	if h == 0 || c.err != nil {
		return nil
	}
	wc, err := c.s.TakeWriter(h)
	if err != nil {
		c.err = err
		return nil
	}
	c.taken = append(c.taken, wc)
	return wc
}

func (c *Conv) ReaderHandle(rc io.ReadCloser) uint32 {
	if rc == nil {
		return 0
	}
	if c.err != nil {
		_ = rc.Close()
		return 0
	}
	h, err := c.s.RegisterReader(rc)
	return c.registeredHandle(h, err)
}

func (c *Conv) WriterHandle(wc io.WriteCloser) uint32 {
	if wc == nil {
		return 0
	}
	if c.err != nil {
		_ = wc.Close()
		return 0
	}
	h, err := c.s.RegisterWriter(wc)
	return c.registeredHandle(h, err)
}

func (c *Conv) registeredHandle(h uint32, err error) uint32 {
	if err != nil {
		c.err = err
		return 0
	}
	c.registered = append(c.registered, h)
	return h
}

// Fail records an error from outside the conversion (the service's), so later outputs are closed, not registered.
func (c *Conv) Fail(err error) {
	if c.err == nil {
		c.err = err
	}
}

// Err reports the first error. On success, taken streams now belong to the service and registered ones to the
// plugin. On an error, both are closed: the service did not run, or the plugin will not get the handles.
func (c *Conv) Err() error {
	if c.err != nil {
		for _, t := range c.taken {
			_ = t.Close()
		}
		for _, h := range c.registered {
			c.s.Release(h)
		}
	}
	c.taken, c.registered = nil, nil
	return c.err
}
