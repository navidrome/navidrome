//go:build !wasip1

package stream

import (
	"errors"
	"io"
)

var errNotWasm = errors.New("streams are only available inside a Navidrome plugin")

type Stream struct{ handle uint32 }

func New(handle uint32) *Stream { return &Stream{handle: handle} }

func (s *Stream) Handle() uint32 { return s.handle }

func (s *Stream) Read([]byte) (int, error) { return 0, errNotWasm }

func (s *Stream) Write([]byte) (int, error) { return 0, errNotWasm }

func (s *Stream) Close() error { return errNotWasm }

func (s *Stream) releaseBuffer() {}

func Copy(dst, src *Stream) (int64, error) { return 0, errNotWasm }

var _ io.ReadWriteCloser = (*Stream)(nil)
