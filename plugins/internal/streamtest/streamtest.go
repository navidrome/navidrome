// Package streamtest is a host service used only by tests, to exercise the code ndpgen generates for streams.
package streamtest

import (
	"context"
	"io"
)

//go:generate sh generate.sh

// WrapRequest gives Body a custom JSON name so the Rust client's serde rename is exercised.
type WrapRequest struct {
	Prefix string        `json:"prefix"`
	Body   io.ReadCloser `json:"payload"`
}

type WrapResponse struct {
	Size int64         `json:"size"`
	Body io.ReadCloser `json:"body"`
}

//nd:hostservice name=StreamTest permission=streamtest
type StreamTestService interface {
	//nd:hostfunc
	Open(ctx context.Context, text string) (io.ReadCloser, error)
	//nd:hostfunc
	Create(ctx context.Context, name string) (io.WriteCloser, error)
	//nd:hostfunc
	Drain(ctx context.Context, in io.ReadCloser) (string, error)
	//nd:hostfunc
	Wrap(ctx context.Context, wrap WrapRequest) (*WrapResponse, error)
	//nd:hostfunc
	Block(ctx context.Context) (io.ReadCloser, error)
	//nd:hostfunc
	Echo(ctx context.Context, text string) (string, error)
}
