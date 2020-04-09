package engine

import (
	"fmt"
	"path/filepath"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/log"
	"github.com/djherbis/fscache"
	"github.com/dustin/go-humanize"
)

func newFileCache(name, cacheSize, cacheFolder string, maxItems int) (fscache.Cache, error) {
	size, err := humanize.ParseBytes(cacheSize)
	if err != nil {
		size = consts.DefaultCacheSize
	}
	lru := fscache.NewLRUHaunter(maxItems, int64(size), consts.DefaultCacheCleanUpInterval)
	h := fscache.NewLRUHaunterStrategy(lru)
	cacheFolder = filepath.Join(conf.Server.DataFolder, cacheFolder)
	log.Info(fmt.Sprintf("Creating %s cache", name), "path", cacheFolder, "maxSize", humanize.Bytes(size))
	fs, err := fscache.NewFs(cacheFolder, 0755)
	if err != nil {
		return nil, err
	}
	return fscache.NewCacheWithHaunter(fs, h)
}
