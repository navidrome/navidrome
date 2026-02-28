package lyrics

import (
	"context"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// Lyrics can fetch lyrics for a media file.
type Lyrics interface {
	GetLyrics(ctx context.Context, mf *model.MediaFile) (model.LyricList, error)
}

// PluginLoader discovers and loads lyrics provider plugins.
type PluginLoader interface {
	LoadLyricsProvider(name string) (Lyrics, bool)
}

type lyricsService struct {
	pluginLoader PluginLoader
}

// NewLyrics creates a new lyrics service. pluginLoader may be nil if no plugin
// system is available.
func NewLyrics(pluginLoader PluginLoader) Lyrics {
	return &lyricsService{pluginLoader: pluginLoader}
}

// GetLyrics returns lyrics for the given media file, trying sources in the
// order specified by conf.Server.LyricsPriority.
func (l *lyricsService) GetLyrics(ctx context.Context, mf *model.MediaFile) (model.LyricList, error) {
	var lyricsList model.LyricList
	var err error

	for pattern := range strings.SplitSeq(strings.ToLower(conf.Server.LyricsPriority), ",") {
		pattern = strings.TrimSpace(pattern)
		switch {
		case pattern == "embedded":
			lyricsList, err = fromEmbedded(ctx, mf)
		case strings.HasPrefix(pattern, "."):
			lyricsList, err = fromExternalFile(ctx, mf, pattern)
		default:
			lyricsList, err = l.fromPlugin(ctx, mf, pattern)
		}

		if err != nil {
			log.Error(ctx, "error getting lyrics", "source", pattern, err)
		}

		if len(lyricsList) > 0 {
			return lyricsList, nil
		}
	}

	return nil, nil
}
