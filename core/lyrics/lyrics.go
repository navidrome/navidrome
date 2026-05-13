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

// BatchLyrics can resolve lyrics across multiple candidate media files while
// still honoring the configured source priority globally.
type BatchLyrics interface {
	GetLyricsForMediaFiles(ctx context.Context, mediaFiles []model.MediaFile) (model.LyricList, error)
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
	return l.getLyricsForCandidates(ctx, []*model.MediaFile{mf})
}

// GetLyricsForMediaFiles resolves lyrics across duplicate media files while
// preserving the configured source priority across the full candidate set.
func (l *lyricsService) GetLyricsForMediaFiles(ctx context.Context, mediaFiles []model.MediaFile) (model.LyricList, error) {
	candidates := make([]*model.MediaFile, 0, len(mediaFiles))
	for i := range mediaFiles {
		candidates = append(candidates, &mediaFiles[i])
	}
	return l.getLyricsForCandidates(ctx, candidates)
}

func (l *lyricsService) getLyricsForCandidates(ctx context.Context, mediaFiles []*model.MediaFile) (model.LyricList, error) {
	for pattern := range strings.SplitSeq(conf.Server.LyricsPriority, ",") {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}

		for _, mf := range mediaFiles {
			if mf == nil {
				continue
			}

			lyricsList, err := l.getLyricsFromSource(ctx, mf, pattern)
			if err != nil {
				log.Error(ctx, "error getting lyrics", "source", pattern, err)
				continue
			}

			if len(lyricsList) > 0 {
				return lyricsList, nil
			}
		}
	}

	return nil, nil
}

func (l *lyricsService) getLyricsFromSource(ctx context.Context, mf *model.MediaFile, pattern string) (model.LyricList, error) {
	switch {
	case strings.EqualFold(pattern, "embedded"):
		return fromEmbedded(ctx, mf)
	case strings.HasPrefix(pattern, "."):
		return fromExternalFile(ctx, mf, strings.ToLower(pattern))
	default:
		return l.fromPlugin(ctx, mf, pattern)
	}
}
