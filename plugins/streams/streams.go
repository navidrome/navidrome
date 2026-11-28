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
	// Hold counts c as an open stream and closes it when the call ends, unless release ran first.
	Hold(c io.Closer) (release func(), err error)
}

type (
	streamsKey  struct{}
	detachedKey struct{}
)

func WithStreams(ctx context.Context, s Streams) context.Context {
	return context.WithValue(ctx, streamsKey{}, s)
}

func From(ctx context.Context) Streams {
	s, _ := ctx.Value(streamsKey{}).(Streams)
	return s
}

// Hold holds c in the call's streams; with no call in progress, it closes c and fails.
func Hold(ctx context.Context, c io.Closer) (release func(), err error) {
	s := From(ctx)
	if s == nil {
		_ = c.Close()
		return nil, errNoCall
	}
	return s.Hold(c)
}

// WithDetached stores base, the call's ctx before Extism wrapped it, for Detached.
func WithDetached(ctx, base context.Context) context.Context {
	return context.WithValue(ctx, detachedKey{}, context.WithoutCancel(base))
}

// Detached returns a ctx for work that outlives the call. It never derives from the host-function ctx,
// which holds the plugin instance and would keep it alive.
func Detached(ctx context.Context) context.Context {
	if d, ok := ctx.Value(detachedKey{}).(context.Context); ok {
		return d
	}
	return context.WithoutCancel(ctx)
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
	s := From(ctx)
	if s == nil {
		return &Conv{err: errNoCall}
	}
	return &Conv{s: s}
}

func (c *Conv) Reader(h uint32) io.ReadCloser { return take(c, h, Streams.TakeReader) }

func (c *Conv) Writer(h uint32) io.WriteCloser { return take(c, h, Streams.TakeWriter) }

func (c *Conv) ReaderHandle(rc io.ReadCloser) uint32 { return register(c, rc, Streams.RegisterReader) }

func (c *Conv) WriterHandle(wc io.WriteCloser) uint32 { return register(c, wc, Streams.RegisterWriter) }

// take and register get method expressions, not method values: c.s is nil when there is no call.
func take[T io.Closer](c *Conv, h uint32, f func(Streams, uint32) (T, error)) T {
	var zero T
	if h == 0 || c.s == nil {
		return zero
	}
	v, err := f(c.s, h)
	if err != nil {
		c.Fail(err)
		return zero
	}
	// Taken even after an error: the plugin handed it over, so Err must close it.
	c.taken = append(c.taken, v)
	if c.err != nil {
		return zero
	}
	return v
}

func register[T io.Closer](c *Conv, v T, f func(Streams, T) (uint32, error)) uint32 {
	if any(v) == nil {
		return 0
	}
	if c.err != nil {
		_ = v.Close()
		return 0
	}
	h, err := f(c.s, v)
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

// Err reports the first error. On success, taken streams now belong to the service. On an error, taken and
// registered streams are closed: the service did not run, or the plugin will not get the handles.
func (c *Conv) Err() error {
	if c.err != nil {
		for _, t := range c.taken {
			_ = t.Close()
		}
		c.Release()
	}
	c.taken = nil
	return c.err
}

// Release closes the streams registered so far, for a response that never reaches the plugin.
func (c *Conv) Release() {
	for _, h := range c.registered {
		c.s.Release(h)
	}
	c.registered = nil
}
