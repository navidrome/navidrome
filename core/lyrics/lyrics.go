package lyrics

import (
	"context"
	"errors"
	"strings"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
)

// maxLegacyLyricsCandidates bounds the duplicate window scanned by the legacy
// artist/title lookup, so source-priority resolution can still reach older
// matches without turning it into an unbounded table scan.
const maxLegacyLyricsCandidates = 10

// Provider fetches lyrics for a single media file. It is the contract
// implemented by individual lyrics sources, such as plugins.
type Provider interface {
	GetLyrics(ctx context.Context, mf *model.MediaFile) (model.LyricList, error)
}

// Lyrics resolves lyrics for media files, honoring the configured source
// priority.
type Lyrics interface {
	Provider
	GetLyricsVariants(ctx context.Context, mf *model.MediaFile, allSources bool) (model.LyricList, error)
	GetLyricsByArtistTitle(ctx context.Context, artist, title string) (model.LyricList, error)
}

// PluginLoader discovers and loads lyrics provider plugins.
type PluginLoader interface {
	LoadLyricsProvider(name string) (Provider, bool)
}

type lyricsService struct {
	ds           model.DataStore
	pluginLoader PluginLoader
}

// NewLyrics creates a new lyrics service. pluginLoader may be nil if no plugin
// system is available.
func NewLyrics(ds model.DataStore, pluginLoader PluginLoader) Lyrics {
	return &lyricsService{ds: ds, pluginLoader: pluginLoader}
}

// GetLyrics returns lyrics for the given media file, trying sources in the
// order specified by conf.Server.LyricsPriority.
func (l *lyricsService) GetLyrics(ctx context.Context, mf *model.MediaFile) (model.LyricList, error) {
	return l.getLyricsForCandidates(ctx, []*model.MediaFile{mf})
}

// GetLyricsVariants includes every configured local source. The initial lookup
// retains priority short-circuiting for plugins; allSources also queries the
// remaining plugins, for background discovery while clients display lyrics.
// Discovery can return both variants and an error when only some sources succeed.
func (l *lyricsService) GetLyricsVariants(ctx context.Context, mf *model.MediaFile, allSources bool) (model.LyricList, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	patterns := []string{}
	seen := map[string]bool{}
	for pattern := range strings.SplitSeq(conf.Server.LyricsPriority, ",") {
		pattern = strings.TrimSpace(pattern)
		if pattern != "" && !seen[pattern] {
			patterns = append(patterns, pattern)
			seen[pattern] = true
		}
	}

	results := make([]sourceResult, len(patterns))
	var discoveryErr error
	if allSources {
		results, discoveryErr = l.discoverSources(ctx, mf, patterns)
	} else {
		found := false
		for i, pattern := range patterns {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if found && !isLocalSource(pattern) {
				continue
			}
			list, err := l.getLyricsFromSource(ctx, mf, pattern)
			results[i] = sourceResult{lyrics: list, err: err}
			found = found || (err == nil && len(list) > 0)
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var variants model.LyricList
	pluginErr := discoveryErr
	partialErr := discoveryErr
	for i, entry := range results {
		if entry.err != nil {
			partialErr = errors.Join(partialErr, entry.err)
			log.Error(ctx, "error getting lyrics", "source", patterns[i], entry.err)
			if !isLocalSource(patterns[i]) {
				pluginErr = errors.Join(pluginErr, entry.err)
			}
			continue
		}
		variants = append(variants, entry.lyrics...)
	}
	if len(variants) > 0 {
		if allSources {
			return variants, partialErr
		}
		return variants, nil
	}
	return nil, pluginErr
}

// Results retain source priority independently of completion order.
type sourceResult struct {
	index  int
	lyrics model.LyricList
	err    error
}

func (l *lyricsService) discoverSources(ctx context.Context, mf *model.MediaFile, patterns []string) ([]sourceResult, error) {
	// Match the plugin host's normal call budget; providers need no special
	// caching or fast-response protocol to participate in source discovery.
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	results := make([]sourceResult, len(patterns))
	completed := make(chan sourceResult, len(patterns))
	for i, pattern := range patterns {
		go func() {
			list, err := l.getLyricsFromSource(ctx, mf, pattern)
			completed <- sourceResult{index: i, lyrics: list, err: err}
		}()
	}
	for remaining := len(patterns); remaining > 0; remaining-- {
		select {
		case entry := <-completed:
			results[entry.index] = entry
		case <-ctx.Done():
			// Drain completed successes before returning partial results.
			// Buffered sends let late providers exit after cancellation.
			for {
				select {
				case entry := <-completed:
					results[entry.index] = entry
				default:
					return results, ctx.Err()
				}
			}
		}
	}
	return results, nil
}

func isLocalSource(pattern string) bool {
	return strings.EqualFold(pattern, "embedded") || strings.HasPrefix(pattern, ".")
}

// GetLyricsByArtistTitle resolves lyrics for the legacy artist/title lookup,
// scanning a bounded window of duplicate matches so source priority still wins
// across them.
func (l *lyricsService) GetLyricsByArtistTitle(ctx context.Context, artist, title string) (model.LyricList, error) {
	opts := songsByArtistTitleWithLyricsFirst(artist, title)
	opts.Max = maxLegacyLyricsCandidates
	mediaFiles, err := l.ds.MediaFile().GetAll(ctx, opts)
	if err != nil {
		return nil, err
	}
	if len(mediaFiles) == 0 {
		return nil, nil
	}
	candidates := make([]*model.MediaFile, 0, len(mediaFiles))
	for i := range mediaFiles {
		candidates = append(candidates, &mediaFiles[i])
	}
	return l.getLyricsForCandidates(ctx, candidates)
}

func songsByArtistTitleWithLyricsFirst(artist, title string) model.QueryOptions {
	return model.QueryOptions{
		Sort:  "lyrics, updated_at",
		Order: "desc",
		Filters: And{
			Eq{"missing": false},
			Eq{"title": title},
			Or{
				persistence.Exists("json_tree(participants, '$.albumartist')", Eq{"value": artist}),
				persistence.Exists("json_tree(participants, '$.artist')", Eq{"value": artist}),
			},
		},
	}
}

func (l *lyricsService) getLyricsForCandidates(ctx context.Context, mediaFiles []*model.MediaFile) (model.LyricList, error) {
	var pluginErr error
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
				if !strings.EqualFold(pattern, "embedded") && !strings.HasPrefix(pattern, ".") {
					pluginErr = errors.Join(pluginErr, err)
				}
				continue
			}

			if len(lyricsList) > 0 {
				return lyricsList, nil
			}
		}
	}

	return nil, pluginErr
}

func (l *lyricsService) getLyricsFromSource(ctx context.Context, mf *model.MediaFile, pattern string) (model.LyricList, error) {
	switch {
	case strings.EqualFold(pattern, "embedded"):
		return fromEmbedded(ctx, mf)
	case strings.HasPrefix(pattern, "."):
		return fromExternalFile(ctx, mf, pattern)
	default:
		return l.fromPlugin(ctx, mf, pattern)
	}
}
