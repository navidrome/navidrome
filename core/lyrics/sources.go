package lyrics

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/navidrome/navidrome/core/storage"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/ioutils"
	"golang.org/x/text/unicode/norm"
)

func fromEmbedded(ctx context.Context, mf *model.MediaFile) (model.LyricList, error) {
	if mf.Lyrics != "" {
		log.Trace(ctx, "embedded lyrics found in file", "title", mf.Title)
		return mf.StructuredLyrics()
	}

	log.Trace(ctx, "no embedded lyrics for file", "path", mf.Title)

	return nil, nil
}

// fromExternalFile reads lyrics from the file sharing mf's base name and ending with suffix.
// folderFiles lists the lyrics files in mf's folder: a file missing from it is not opened.
// If nil, the filesystem is always probed. The list is matched ignoring case and Unicode
// normalization, as the filesystem may do, but the file is still opened by its expected name
// (see openSidecar).
func fromExternalFile(ctx context.Context, mf *model.MediaFile, folderFiles []string, suffix string) (model.LyricList, error) {
	ext := path.Ext(mf.Path)
	sidecarRelPath := mf.Path[0:len(mf.Path)-len(ext)] + suffix
	name := norm.NFC.String(path.Base(sidecarRelPath))
	if folderFiles != nil && !slices.ContainsFunc(folderFiles, func(f string) bool { return strings.EqualFold(norm.NFC.String(f), name) }) {
		return nil, nil
	}
	ctx = log.NewContext(ctx, "file", sidecarRelPath)

	store, err := storage.For(mf.LibraryPath)
	if err != nil {
		return nil, fmt.Errorf("getting storage for library: %w", err)
	}
	fsys, err := store.FS()
	if err != nil {
		return nil, fmt.Errorf("opening library filesystem: %w", err)
	}

	f, openedPath, err := openSidecar(fsys, sidecarRelPath)
	if errors.Is(err, fs.ErrNotExist) {
		log.Trace(ctx, "no lyrics found at path")
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	defer f.Close()
	// Log the name that is really on disk, so a parse error points at the right file.
	ctx = log.NewContext(ctx, "file", openedPath)

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
	return list, nil
}

// openSidecar opens the sidecar lyrics file and returns the path it opened. When the
// exact path is missing, it retries with the NFD and NFC forms of the file name: the
// sidecar can be in another Unicode form than the stored track path (macOS often
// writes NFD, Linux and Windows keep NFC). The folders are kept as stored, because
// the scanner read them from disk.
func openSidecar(fsys fs.FS, relPath string) (fs.File, string, error) {
	f, err := fsys.Open(relPath)
	if !errors.Is(err, fs.ErrNotExist) {
		return f, relPath, err
	}

	dir, base := path.Split(relPath)
	for _, form := range []norm.Form{norm.NFD, norm.NFC} {
		altPath := dir + form.String(base)
		if altPath == relPath {
			continue
		}
		if altFile, altErr := fsys.Open(altPath); !errors.Is(altErr, fs.ErrNotExist) {
			return altFile, altPath, altErr
		}
	}
	return f, relPath, err
}

// fromPlugin attempts to load lyrics from a plugin with the given name. Results are cached,
// misses included: clients ask for every track in a queue, and plugin lookups are slow.
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

	key := pluginKey{plugin: pluginName, mediaFileID: mf.ID}
	return l.pluginCache.GetWithLoader(key, func(pluginKey) (model.LyricList, time.Duration, error) {
		// Concurrent callers share one load, so one cancelled request must not abort it for the others.
		loadCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), lyricsLoadTimeout)
		defer cancel()
		lyricsList, err := provider.GetLyrics(loadCtx, mf)
		if err != nil {
			// Cached as a miss: some plugins report "no lyrics" as an error.
			log.Error(ctx, "Error getting lyrics from plugin", "plugin", pluginName, err)
			return nil, 0, nil
		}
		if len(lyricsList) > 0 {
			log.Trace(ctx, "Retrieved lyrics from plugin", "plugin", pluginName, "count", len(lyricsList))
		}
		return lyricsList, 0, nil
	})
}
