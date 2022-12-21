package tests

import (
	"context"
	"io"
	"strings"
	"sync"

	"github.com/navidrome/navidrome/utils"
)

func NewMockFFmpeg(data string) *MockFFmpeg {
	return &MockFFmpeg{Reader: strings.NewReader(data)}
}

type MockFFmpeg struct {
	io.Reader
	lock   sync.Mutex
	closed utils.AtomicBool
	Error  error
}

func (ff *MockFFmpeg) Transcode(ctx context.Context, cmd, path string, maxBitRate int) (f io.ReadCloser, err error) {
	if ff.Error != nil {
		return nil, ff.Error
	}
	return ff, nil
}

func (ff *MockFFmpeg) ExtractImage(ctx context.Context, path string) (io.ReadCloser, error) {
	if ff.Error != nil {
		return nil, ff.Error
	}
	return ff, nil
}

func (ff *MockFFmpeg) Read(p []byte) (n int, err error) {
	ff.lock.Lock()
	defer ff.lock.Unlock()
	return ff.Reader.Read(p)
}

func (ff *MockFFmpeg) Close() error {
	ff.closed.Set(true)
	return nil
}

func (ff *MockFFmpeg) IsClosed() bool {
	return ff.closed.Get()
}
