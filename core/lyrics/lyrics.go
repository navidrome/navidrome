package lyrics

import (
	"context"
	"slices"
	"strings"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/utils/cache"
)

// maxLegacyLyricsCandidates bounds the duplicate window scanned by the legacy
// artist/title lookup, so source-priority resolution can still reach older
// matches without turning it into an unbounded table scan.
const maxLegacyLyricsCandidates = 10

const (
	lyricsCacheSize   = 1000
	lyricsCacheTTL    = 5 * time.Minute
	lyricsLoadTimeout = time.Minute
)

// Provider fetches lyrics for a single media file. It is the contract
// implemented by individual lyrics sources, such as plugins.
type Provider interface {
	GetLyrics(ctx context.Context, mf *model.MediaFile) (model.LyricList, error)
}

// Lyrics resolves lyrics for media files, honoring the configured source
// priority.
type Lyrics interface {
	Provider
	GetLyricsByArtistTitle(ctx context.Context, artist, title string) (model.LyricList, error)
}

// PluginLoader discovers and loads lyrics provider plugins.
type PluginLoader interface {
	LoadLyricsProvider(name string) (Provider, bool)
}

type lyricsService struct {
	ds           model.DataStore
	pluginLoader PluginLoader
	pluginCache  cache.SimpleCache[pluginKey, model.LyricList]
}

type pluginKey struct {
	plugin      string
	mediaFileID string
}

// NewLyrics creates a new lyrics service. pluginLoader may be nil if no plugin
// system is available.
func NewLyrics(ds model.DataStore, pluginLoader PluginLoader) Lyrics {
	return &lyricsService{
		ds:           ds,
		pluginLoader: pluginLoader,
		pluginCache: cache.NewSimpleCache[pluginKey, model.LyricList](cache.Options{
			SizeLimit:  lyricsCacheSize,
			DefaultTTL: lyricsCacheTTL,
		}),
	}
}

// GetLyrics returns lyrics for the given media file, trying sources in the
// order specified by conf.Server().LyricsPriority.
func (l *lyricsService) GetLyrics(ctx context.Context, mf *model.MediaFile) (model.LyricList, error) {
	return l.getLyricsForCandidates(ctx, []*model.MediaFile{mf})
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
	folders := make(map[string][]string)
	for _, pattern := range priorityPatterns() {
		for _, mf := range mediaFiles {
			if mf == nil {
				continue
			}

			lyricsList, err := l.getLyricsFromSource(ctx, mf, pattern, folders)
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

func (l *lyricsService) getLyricsFromSource(ctx context.Context, mf *model.MediaFile, pattern string, folders map[string][]string) (model.LyricList, error) {
	switch {
	case strings.EqualFold(pattern, "embedded"):
		return fromEmbedded(ctx, mf)
	case isFileSuffix(pattern):
		return fromExternalFile(ctx, mf, l.lyricsFilesOf(ctx, mf, folders), pattern)
	default:
		return l.fromPlugin(ctx, mf, pattern)
	}
}

// lyricsFilesOf returns nil when the folder was not scanned for lyrics files yet, so the caller probes the filesystem.
func (l *lyricsService) lyricsFilesOf(ctx context.Context, mf *model.MediaFile, folders map[string][]string) []string {
	files, ok := folders[mf.FolderID]
	if !ok {
		if folder, err := l.ds.Folder().Get(ctx, mf.FolderID); err != nil {
			log.Warn(ctx, "Error loading folder of media file. Looking for lyrics files in the filesystem", "path", mf.Path, err)
		} else {
			files = folder.LyricsFiles
		}
		folders[mf.FolderID] = files
	}
	return files
}

func priorityPatterns() []string {
	var res []string
	for p := range strings.SplitSeq(conf.Server().LyricsPriority, ",") {
		if p = strings.TrimSpace(p); p != "" {
			res = append(res, p)
		}
	}
	return res
}

func isFileSuffix(pattern string) bool {
	return strings.HasPrefix(pattern, ".")
}

// FileSuffixes returns the sidecar suffixes in LyricsPriority, lowercased and sorted so that
// reordering or recasing them does not change the scanner's folder hashes.
func FileSuffixes() []string {
	var res []string
	for _, p := range priorityPatterns() {
		if isFileSuffix(p) {
			res = append(res, strings.ToLower(p))
		}
	}
	slices.Sort(res)
	return slices.Compact(res)
}
