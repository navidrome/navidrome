package plugins

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/plugins/host"
)

const storageMount = "/storage"

type storageServiceImpl struct {
	root *os.Root
}

func getHostStoragePath(pluginName string) string {
	return filepath.Join(conf.Server().DataFolder.String(), "plugins", pluginName, "storage")
}

func newStorageService(pluginName string) (*storageServiceImpl, error) {
	dataDir := getHostStoragePath(pluginName)
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("creating plugin data directory: %w", err)
	}
	root, err := os.OpenRoot(dataDir)
	if err != nil {
		return nil, fmt.Errorf("opening plugin data directory: %w", err)
	}
	return &storageServiceImpl{root: root}, nil
}

func (s *storageServiceImpl) GetStoragePath(ctx context.Context) string {
	return storageMount
}

func (s *storageServiceImpl) Open(_ context.Context, path string) (io.ReadCloser, error) {
	p, err := storageRelPath(path)
	if err != nil {
		return nil, err
	}
	return s.root.Open(p)
}

func (s *storageServiceImpl) Create(_ context.Context, path string) (io.WriteCloser, error) {
	p, err := storageRelPath(path)
	if err != nil {
		return nil, err
	}
	if err := s.root.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return nil, err
	}
	return s.root.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
}

// Close closes the storage root; files already opened from it stay usable.
func (s *storageServiceImpl) Close() error {
	return s.root.Close()
}

// storageRelPath accepts only paths that stay inside the storage folder lexically; os.Root guards symlinks.
func storageRelPath(path string) (string, error) {
	p := filepath.FromSlash(path)
	if path == "" || !filepath.IsLocal(p) {
		return "", fmt.Errorf("invalid storage path %q: must be relative and inside the plugin storage", path)
	}
	return p, nil
}

var _ host.StorageService = (*storageServiceImpl)(nil)
