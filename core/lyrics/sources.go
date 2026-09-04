package lyrics

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"

	"github.com/navidrome/navidrome/core/storage"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/ioutils"
)

func fromEmbedded(ctx context.Context, mf *model.MediaFile) (model.LyricList, error) {
	if mf.Lyrics != "" {
		log.Trace(ctx, "embedded lyrics found in file", "title", mf.Title)
		list, err := mf.StructuredLyrics()
		return withLyricsSource(list, model.LyricsSource{Type: model.LyricsSourceEmbedded}), err
	}

	log.Trace(ctx, "no embedded lyrics for file", "path", mf.Title)

	return nil, nil
}

func fromExternalFile(ctx context.Context, mf *model.MediaFile, suffix string) (model.LyricList, error) {
	ext := path.Ext(mf.Path)
	sidecarRelPath := mf.Path[0:len(mf.Path)-len(ext)] + suffix
	ctx = log.NewContext(ctx, "file", sidecarRelPath)

	store, err := storage.For(mf.LibraryPath)
	if err != nil {
		return nil, fmt.Errorf("getting storage for library: %w", err)
	}
	fsys, err := store.FS()
	if err != nil {
		return nil, fmt.Errorf("opening library filesystem: %w", err)
	}

	f, err := fsys.Open(sidecarRelPath)
	if errors.Is(err, fs.ErrNotExist) {
		log.Trace(ctx, "no lyrics found at path")
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	defer f.Close()

	contents, err := io.ReadAll(ioutils.UTF8Reader(f))
	if err != nil {
		return nil, err
	}

	list, err := model.ParseLyrics(ctx, suffix, "xxx", contents)
	if err != nil {
		log.Error(ctx, "error parsing external lyric file", err)
		return nil, err
	}

	if len(list) == 0 {
		log.Trace(ctx, "empty lyrics from external file")
		return nil, nil
	}

	log.Trace(ctx, "retrieved lyrics from external file")
	return withLyricsSource(list, model.LyricsSource{
		Type:   model.LyricsSourceSidecar,
		Format: strings.ToLower(strings.TrimPrefix(suffix, ".")),
	}), nil
}

// fromPlugin attempts to load lyrics from a plugin with the given name.
func (l *lyricsService) fromPlugin(ctx context.Context, mf *model.MediaFile, pluginName string) (model.LyricList, error) {
	if l.pluginLoader == nil {
		log.Debug(ctx, "Invalid lyric source", "source", pluginName)
		return nil, nil
	}

	provider, ok := l.pluginLoader.LoadLyricsProvider(pluginName)
	if !ok {
		log.Warn(ctx, "Lyrics plugin not found", "plugin", pluginName)
		return nil, nil
	}

	lyricsList, err := provider.GetLyrics(ctx, mf)
	if err != nil {
		return nil, err
	}

	if len(lyricsList) > 0 {
		log.Trace(ctx, "Retrieved lyrics from plugin", "plugin", pluginName, "count", len(lyricsList))
	}
	return withLyricsSource(lyricsList, model.LyricsSource{
		Type: model.LyricsSourcePlugin,
		Name: pluginName,
	}), nil
}

// withLyricsSource copies the lyric entries before adding defaults so resolver
// metadata does not mutate cached embedded lyrics or a provider-owned result.
func withLyricsSource(list model.LyricList, defaults model.LyricsSource) model.LyricList {
	if len(list) == 0 {
		return list
	}

	result := make(model.LyricList, len(list))
	copy(result, list)
	for i := range result {
		source := defaults
		if result[i].Source != nil {
			source = *result[i].Source
			if source.Type == "" {
				source.Type = defaults.Type
			}
			if source.Name == "" {
				source.Name = defaults.Name
			}
			if source.Provider == "" {
				source.Provider = defaults.Provider
			}
			if source.Format == "" {
				source.Format = defaults.Format
			}
		}
		result[i].Source = &source
	}
	return result
}
