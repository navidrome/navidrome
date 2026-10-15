package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/plugins/host"
)

const storageMount = "/storage"

type storageServiceImpl struct{}

func getHostStoragePath(pluginName string) string {
	return filepath.Join(conf.Server.DataFolder.String(), "plugins", pluginName, "storage")
}

func newStorageService(pluginName string) (host.StorageService, error) {
	dataDir := getHostStoragePath(pluginName)
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("creating plugin data directory: %w", err)
	}

	return &storageServiceImpl{}, nil
}

func (s *storageServiceImpl) GetStoragePath(ctx context.Context) string {
	return storageMount
}

var _ host.StorageService = (*storageServiceImpl)(nil)
