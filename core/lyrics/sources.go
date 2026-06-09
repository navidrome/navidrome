package lyrics

import (
	"context"
	"errors"
	"os"
	"path"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/ioutils"
)

func fromEmbedded(ctx context.Context, mf *model.MediaFile) (model.LyricList, error) {
	if mf.Lyrics != "" {
		log.Trace(ctx, "embedded lyrics found in file", "title", mf.Title)
		return mf.StructuredLyrics()
	}

	log.Trace(ctx, "no embedded lyrics for file", "path", mf.Title)

	return nil, nil
}

func fromExternalFile(ctx context.Context, mf *model.MediaFile, suffix string) (model.LyricList, error) {
	basePath := mf.AbsolutePath()
	ext := path.Ext(basePath)

	externalLyric := basePath[0:len(basePath)-len(ext)] + suffix

	contents, err := ioutils.UTF8ReadFile(externalLyric)
	if errors.Is(err, os.ErrNotExist) {
		log.Trace(ctx, "no lyrics found at path", "path", externalLyric)
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	lyrics, err := model.ToLyrics("xxx", string(contents))
	if err != nil {
		log.Error(ctx, "error parsing lyric external file", "path", externalLyric, err)
		return nil, err
	} else if lyrics == nil {
		log.Trace(ctx, "empty lyrics from external file", "path", externalLyric)
		return nil, nil
	}

	log.Trace(ctx, "retrieved lyrics from external file", "path", externalLyric)

	return model.LyricList{*lyrics}, nil
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
	return lyricsList, nil
}
