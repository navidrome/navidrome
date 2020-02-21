package engine

import (
	"path/filepath"
	"time"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/engine/ffmpeg"
	"github.com/google/wire"
	"gopkg.in/djherbis/fscache.v0"
)

var Set = wire.NewSet(
	NewBrowser,
	NewCover,
	NewListGenerator,
	NewPlaylists,
	NewRatings,
	NewScrobbler,
	NewSearch,
	NewNowPlayingRepository,
	NewUsers,
	NewMediaStreamer,
	ffmpeg.New,
	NewTranscodingCache,
)

func NewTranscodingCache() (fscache.Cache, error) {
	lru := fscache.NewLRUHaunter(0, conf.Server.MaxTranscodingCacheSize, 30*time.Second)
	h := fscache.NewLRUHaunterStrategy(lru)
	cacheFolder := filepath.Join(conf.Server.DataFolder, consts.CacheDir)
	fs, err := fscache.NewFs(cacheFolder, 0755)
	if err != nil {
		return nil, err
	}
	return fscache.NewCacheWithHaunter(fs, h)
}
