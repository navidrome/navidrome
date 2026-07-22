package artwork

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"io/fs"
	"net/url"
	"os"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/model"
)

// resolution is one attempted acquisition outcome for an entity.
type resolution struct {
	reader     io.ReadCloser // nil when no source yielded an image
	source     string        // model.ItemArtwork.Source value: "folder", "embedded", "external", "upload", "generated"
	sourcePath string        // backing library/upload file (folder/upload: the image; embedded: the audio file); "" otherwise
	refMtime   int64         // mtime of sourcePath at resolution time; 0 when no sourcePath
	// external source errored/timed out. With no reader: forces failed (never absent).
	// On a hit: a higher-priority external step failed—serve this, but retry later.
	extError bool
}

// extGateFunc is an alias for the external-step wrapper the worker injects (rate
// limiter + circuit breaker); resolveItem defaults to a plain passthrough.
type extGateFunc = func(func() (io.ReadCloser, string, error)) (io.ReadCloser, string, error)

func passthroughExtGate(f func() (io.ReadCloser, string, error)) (io.ReadCloser, string, error) {
	return f()
}

// resolveItem walks the kind's priority chain and returns the first hit.
func resolveItem(ctx context.Context, ds model.DataStore, prov external.Provider, ffmpeg ffmpeg.FFmpeg, item model.ArtworkQueueItem, extGate extGateFunc) (resolution, error) {
	if extGate == nil {
		extGate = passthroughExtGate
	}
	switch item.ItemKind {
	case "al":
		return resolveAlbum(ctx, ds, prov, ffmpeg, item.ItemID, extGate)
	case "ar":
		return resolveArtist(ctx, ds, prov, ffmpeg, item.ItemID, extGate)
	case "pl":
		return resolvePlaylist(ctx, ds, prov, ffmpeg, item.ItemID, extGate)
	case "ra":
		return resolveRadio(ctx, ds, item.ItemID)
	default:
		return resolution{}, fmt.Errorf("resolveItem: kind %q is not resolvable by the worker", item.ItemKind)
	}
}

// resolveAlbum ports the folder/embedded/external selection from
// reader_album.go, walking conf.Server.CoverArtPriority.
func resolveAlbum(ctx context.Context, ds model.DataStore, prov external.Provider, ffm ffmpeg.FFmpeg, albumID string, extGate extGateFunc) (resolution, error) {
	al, err := ds.Album(ctx).Get(albumID)
	if err != nil {
		return resolution{}, err
	}
	_, imgFiles, _, err := loadAlbumFoldersPaths(ctx, ds, *al)
	if err != nil {
		return resolution{}, err
	}
	lib, err := loadLibraryView(ctx, ds, al.LibraryID)
	if err != nil {
		return resolution{}, err
	}

	var extErr bool
	for pattern := range strings.SplitSeq(strings.ToLower(conf.Server.CoverArtPriority), ",") {
		pattern = strings.TrimSpace(pattern)
		switch {
		case pattern == "embedded":
			if res, ok := resolveEmbedded(ctx, lib, ffm, al.EmbedArtPath); ok {
				res.extError = extErr
				return res, nil
			}
		case pattern == "external":
			if res, ok, isErr := resolveExternalStep(extGate, fromAlbumExternalSource(ctx, *al, prov)); ok {
				return res, nil
			} else if isErr {
				extErr = true
			}
		case len(imgFiles) > 0:
			if res, ok := resolveFolderFile(ctx, lib, imgFiles, pattern); ok {
				res.extError = extErr
				return res, nil
			}
		}
	}
	return resolution{extError: extErr}, nil
}

// resolveArtist ports the upload/folder/external selection from
// reader_artist.go: upload always wins, then conf.Server.ArtistArtPriority.
func resolveArtist(ctx context.Context, ds model.DataStore, prov external.Provider, ffm ffmpeg.FFmpeg, artistID string, extGate extGateFunc) (resolution, error) {
	ar, err := ds.Artist(ctx).Get(artistID)
	if err != nil {
		return resolution{}, err
	}
	if res, ok := resolveLocalFile(ar.UploadedImagePath(), "upload"); ok {
		return res, nil
	}

	// Only consider albums where the artist is the sole album artist, same as reader_artist.go.
	als, err := ds.Album(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.And{
			squirrel.Eq{"album_artist_id": artistID},
			squirrel.Eq{"json_array_length(participants, '$.albumartist')": 1},
		},
	})
	if err != nil {
		return resolution{}, err
	}
	albumPaths, imgFiles, _, err := loadAlbumFoldersPaths(ctx, ds, als...)
	if err != nil {
		return resolution{}, err
	}
	artistFolder, _, err := loadArtistFolder(ctx, ds, als, albumPaths)
	if err != nil {
		return resolution{}, err
	}
	var lib libraryView
	if len(als) > 0 {
		lib, err = loadLibraryView(ctx, ds, als[0].LibraryID)
		if err != nil {
			return resolution{}, err
		}
	}

	var extErr bool
	for pattern := range strings.SplitSeq(strings.ToLower(conf.Server.ArtistArtPriority), ",") {
		pattern = strings.TrimSpace(pattern)
		switch {
		case pattern == "external":
			if res, ok, isErr := resolveExternalStep(extGate, fromArtistExternalResult(ctx, *ar, prov)); ok {
				return res, nil
			} else if isErr {
				extErr = true
			}
		case pattern == "image-folder":
			if res, ok := resolveArtistImageFolder(ar); ok {
				res.extError = extErr
				return res, nil
			}
		case strings.HasPrefix(pattern, "album/"):
			if lib.FS == nil {
				continue
			}
			if res, ok := resolveFolderFile(ctx, lib, imgFiles, strings.TrimPrefix(pattern, "album/")); ok {
				res.extError = extErr
				return res, nil
			}
		default:
			if lib.FS == nil || artistFolder == "" {
				continue
			}
			if res, ok := resolveArtistFolderPattern(ctx, lib, artistFolder, pattern); ok {
				res.extError = extErr
				return res, nil
			}
		}
	}
	return resolution{extError: extErr}, nil
}

// resolvePlaylist ports reader_playlist.go's chain: uploaded image, sidecar,
// ExternalImageURL, then the generated 2x2 grid sourced through resolveAlbum.
func resolvePlaylist(ctx context.Context, ds model.DataStore, prov external.Provider, ffm ffmpeg.FFmpeg, playlistID string, extGate extGateFunc) (resolution, error) {
	pl, err := ds.Playlist(ctx).Get(playlistID)
	if err != nil {
		return resolution{}, err
	}

	var extErr bool
	if res, ok := resolveLocalFile(pl.UploadedImagePath(), "upload"); ok {
		return res, nil
	}
	if res, ok := resolveLocalFile(findPlaylistSidecarPath(ctx, pl.Path), "folder"); ok {
		return res, nil
	}
	if res, ok, isErr := resolveExternalStep(extGate, fromPlaylistExternalSource(ctx, *pl)); ok {
		return res, nil
	} else if isErr {
		extErr = true
	}

	albumIDs, err := ds.Playlist(ctx).Tracks(pl.ID, false).GetAlbumIDs(model.QueryOptions{Max: 4, Sort: "random()"})
	if err != nil {
		return resolution{}, err
	}

	var tiles []image.Image
	var tileErr error // first internal (non-external) tile failure, e.g. album deleted mid-flight
	for _, albumID := range albumIDs {
		res, err := resolveAlbum(ctx, ds, prov, ffm, albumID, extGate)
		if err != nil {
			if tileErr == nil {
				tileErr = err
			}
			continue
		}
		if res.extError {
			extErr = true
		}
		if res.reader == nil {
			continue
		}
		tile, decErr := decodeTile(res.reader)
		res.reader.Close()
		if decErr == nil {
			tiles = append(tiles, tile)
		}
		if len(tiles) == 4 {
			break
		}
	}
	if len(tiles) == 0 {
		// A tile-level failure must never resolve as a clean absent: propagate
		// internal errors, and force extError for external ones.
		if tileErr != nil {
			return resolution{}, fmt.Errorf("resolvePlaylist: sampled album art failed: %w", tileErr)
		}
		return resolution{extError: extErr}, nil
	}
	// Grow to 4 tiles by repeating what we have, mirroring reader_playlist.go's loadTiles.
	switch len(tiles) {
	case 2:
		tiles = append(tiles, tiles[1], tiles[0])
	case 3:
		tiles = append(tiles, tiles[0])
	}
	r, err := assembleTiles(tiles)
	if err != nil {
		return resolution{extError: extErr}, nil //nolint:nilerr // encode failure is a soft "no image", not a resolveItem error
	}
	return resolution{reader: r, source: "generated", extError: extErr}, nil
}

// resolveRadio ports reader_radio.go: only an uploaded image, no fallback.
func resolveRadio(ctx context.Context, ds model.DataStore, radioID string) (resolution, error) {
	r, err := ds.Radio(ctx).Get(radioID)
	if err != nil {
		return resolution{}, err
	}
	res, _ := resolveLocalFile(r.UploadedImagePath(), "upload")
	return res, nil
}

// resolveExternalStep runs an external sourceFunc through extGate, shared by
// resolveAlbum and resolveArtist. ok reports a hit; extErr reports a
// non-not-found error (a not-found is a definitive "no", not a failure).
func resolveExternalStep(extGate extGateFunc, sf func() (io.ReadCloser, string, error)) (res resolution, ok bool, extErr bool) {
	r, path, err := extGate(sf)
	if r != nil {
		return resolution{reader: r, source: "external", sourcePath: path}, true, false
	}
	return resolution{}, false, err != nil && !errors.Is(err, model.ErrNotFound)
}

// fromPlaylistExternalSource mirrors reader_playlist.go's ExternalImageURL step:
// a remote URL (gated) when M3U external art is enabled, else a local file path.
func fromPlaylistExternalSource(ctx context.Context, pl model.Playlist) sourceFunc {
	return func() (io.ReadCloser, string, error) {
		imgURL := pl.ExternalImageURL
		if imgURL == "" {
			return nil, "", nil
		}
		parsed, err := url.Parse(imgURL)
		if err != nil {
			return nil, "", err
		}
		if parsed.Scheme == "http" || parsed.Scheme == "https" {
			if !conf.Server.EnableM3UExternalAlbumArt {
				return nil, "", nil
			}
			return fromURL(ctx, parsed)
		}
		// A missing/unreadable local file is a definitive miss, not a transient
		// failure to retry: swallow the open error and fall through to the grid.
		r, path, _ := fromLocalFile(imgURL)()
		return r, path, nil
	}
}

func resolveEmbedded(ctx context.Context, lib libraryView, ffm ffmpeg.FFmpeg, embedRel string) (resolution, bool) {
	if embedRel == "" {
		return resolution{}, false
	}
	abs := lib.Abs(embedRel)
	for _, sf := range []sourceFunc{fromTag(ctx, lib.FS, embedRel), fromFFmpegTag(ctx, ffm, abs)} {
		if r, _, _ := sf(); r != nil {
			return resolution{reader: r, source: "embedded", sourcePath: abs, refMtime: mtimeViaFS(lib.FS, embedRel)}, true
		}
	}
	return resolution{}, false
}

func resolveFolderFile(ctx context.Context, lib libraryView, imgFiles []string, pattern string) (resolution, bool) {
	r, path, _ := fromExternalFile(ctx, lib.FS, imgFiles, pattern)()
	if r == nil {
		return resolution{}, false
	}
	return resolution{reader: r, source: "folder", sourcePath: lib.Abs(path), refMtime: mtimeViaFS(lib.FS, path)}, true
}

func resolveArtistImageFolder(ar *model.Artist) (resolution, bool) {
	folder := conf.Server.ArtistImageFolder
	if folder == "" {
		return resolution{}, false
	}
	return resolveLocalFile(findImageInArtistFolder(folder, ar.MbzArtistID, ar.Name), "folder")
}

func resolveArtistFolderPattern(ctx context.Context, lib libraryView, artistFolder, pattern string) (resolution, bool) {
	r, path, _ := fromArtistFolder(ctx, lib.FS, lib.absRoot, artistFolder, pattern)()
	if r == nil {
		return resolution{}, false
	}
	return resolution{reader: r, source: "folder", sourcePath: path, refMtime: mtimeOf(path)}, true
}

// resolveLocalFile opens an absolute path directly (uploads, image-folder). A
// missing or unreadable path is "no source", not an error.
func resolveLocalFile(path, source string) (resolution, bool) {
	if path == "" {
		return resolution{}, false
	}
	f, err := os.Open(path)
	if err != nil {
		return resolution{}, false
	}
	return resolution{reader: f, source: source, sourcePath: path, refMtime: mtimeOf(path)}, true
}

func mtimeOf(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.ModTime().Unix()
}

// mtimeViaFS stats through the library FS instead of a joined absolute path,
// since library roots in tests may not be real OS paths (e.g. testfile://).
func mtimeViaFS(fsys fs.FS, name string) int64 {
	if fsys == nil || name == "" {
		return 0
	}
	info, err := fs.Stat(fsys, name)
	if err != nil {
		return 0
	}
	return info.ModTime().Unix()
}

// decodeTile and assembleTiles mirror playlistArtworkReader's createTile/
// createTiledImage, reusing the same rect/fillCenter cropping helpers.
func decodeTile(r io.ReadCloser) (image.Image, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}
	return fillCenter(img, tileSize/2, tileSize/2), nil
}

func assembleTiles(tiles []image.Image) (io.ReadCloser, error) {
	buf := new(bytes.Buffer)
	var err error
	if len(tiles) == 4 {
		rgba := image.NewRGBA(image.Rectangle{Max: image.Point{X: tileSize - 1, Y: tileSize - 1}})
		draw.Draw(rgba, rect(0), tiles[0], image.Point{}, draw.Src)
		draw.Draw(rgba, rect(1), tiles[1], image.Point{}, draw.Src)
		draw.Draw(rgba, rect(2), tiles[2], image.Point{}, draw.Src)
		draw.Draw(rgba, rect(3), tiles[3], image.Point{}, draw.Src)
		err = png.Encode(buf, rgba)
	} else {
		err = png.Encode(buf, tiles[0])
	}
	if err != nil {
		return nil, err
	}
	return io.NopCloser(buf), nil
}
