package tests

import (
	"context"
	"io"
	"strings"
	"sync"
	"sync/atomic"
)

func NewMockFFmpeg(data string) *MockFFmpeg {
	return &MockFFmpeg{Reader: strings.NewReader(data)}
}

type MockFFmpeg struct {
	io.Reader
	lock   sync.Mutex
	closed atomic.Bool
	Error  error
}

func (ff *MockFFmpeg) IsAvailable() bool {
	return true
}

func (ff *MockFFmpeg) Transcode(context.Context, string, string, int, int) (io.ReadCloser, error) {
	if ff.Error != nil {
		return nil, ff.Error
	}
	return ff, nil
}

func (ff *MockFFmpeg) ExtractImage(context.Context, string) (io.ReadCloser, error) {
	if ff.Error != nil {
		return nil, ff.Error
	}
	return ff, nil
}

func (ff *MockFFmpeg) Probe(context.Context, []string) (string, error) {
	if ff.Error != nil {
		return "", ff.Error
	}
	return "", nil
}
func (ff *MockFFmpeg) CmdPath() (string, error) {
	if ff.Error != nil {
		return "", ff.Error
	}
	return "ffmpeg", nil
}

func (ff *MockFFmpeg) Version() string {
	return "1.0"
}

func (ff *MockFFmpeg) Read(p []byte) (n int, err error) {
	ff.lock.Lock()
	defer ff.lock.Unlock()
	return ff.Reader.Read(p)
}

func (ff *MockFFmpeg) Close() error {
	ff.closed.Store(true)
	return nil
}

func (ff *MockFFmpeg) IsClosed() bool {
	return ff.closed.Load()
}
