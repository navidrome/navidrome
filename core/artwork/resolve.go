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
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/agents"
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

// resolveItem walks the kind's priority chain and returns the first hit.
func resolveItem(ctx context.Context, ds model.DataStore, ag *agents.Agents, ffmpeg ffmpeg.FFmpeg, item model.ArtworkQueueItem, gate gateFunc) (resolution, error) {
	return resolveItemMode(ctx, ds, ag, ffmpeg, item, gate, false)
}

// resolveItemLocal resolves using only local sources for the serving path's provisional
// read-through: external steps are skipped and the worker-built playlist grid is not assembled.
func resolveItemLocal(ctx context.Context, ds model.DataStore, ffmpeg ffmpeg.FFmpeg, item model.ArtworkQueueItem) (resolution, error) {
	return resolveItemMode(ctx, ds, nil, ffmpeg, item, denyGate, true)
}

func resolveItemMode(ctx context.Context, ds model.DataStore, ag *agents.Agents, ffmpeg ffmpeg.FFmpeg, item model.ArtworkQueueItem, gate gateFunc, localOnly bool) (resolution, error) {
	if gate == nil {
		gate = passthroughGate
	}
	switch item.ItemKind {
	case "al":
		return resolveAlbum(ctx, ds, ag, ffmpeg, item.ItemID, gate, localOnly)
	case "ar":
		return resolveArtist(ctx, ds, ag, ffmpeg, item.ItemID, gate, localOnly)
	case "pl":
		return resolvePlaylist(ctx, ds, ag, ffmpeg, item.ItemID, gate, localOnly)
	case "ra":
		return resolveRadio(ctx, ds, item.ItemID)
	case "mf":
		return resolveMediaFile(ctx, ds, ffmpeg, item.ItemID)
	default:
		return resolution{}, fmt.Errorf("resolveItem: kind %q is not resolvable by the worker", item.ItemKind)
	}
}

// resolveAlbum ports the folder/embedded/external selection from
// reader_album.go, walking conf.Server.CoverArtPriority.
func resolveAlbum(ctx context.Context, ds model.DataStore, ag *agents.Agents, ffm ffmpeg.FFmpeg, albumID string, gate gateFunc, localOnly bool) (resolution, error) {
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
			if localOnly {
				continue
			}
			if r, name, isErr := fetchAlbumImage(ctx, ag, gate, *al); r != nil {
				return resolution{reader: r, source: "external:" + name}, nil
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
func resolveArtist(ctx context.Context, ds model.DataStore, ag *agents.Agents, ffm ffmpeg.FFmpeg, artistID string, gate gateFunc, localOnly bool) (resolution, error) {
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
			if localOnly {
				continue
			}
			if r, name, isErr := fetchArtistImage(ctx, ag, gate, *ar); r != nil {
				return resolution{reader: r, source: "external:" + name}, nil
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
func resolvePlaylist(ctx context.Context, ds model.DataStore, ag *agents.Agents, ffm ffmpeg.FFmpeg, playlistID string, gate gateFunc, localOnly bool) (resolution, error) {
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
	// A local ExternalImageURL is a file-backed reference: serve it in place (staleness-checked,
	// and available even on the request path). Only http(s) URLs need the gated remote fetch.
	localImg, remoteImg := classifyPlaylistImage(pl.ExternalImageURL)
	if localImg != "" {
		if res, ok := resolveLocalFile(localImg, "folder"); ok {
			return res, nil
		}
	}
	if localOnly {
		// The remote ExternalImageURL fetch and the 2x2 grid are worker-only; a request must
		// not fetch remotely nor sample album art synchronously.
		return resolution{}, nil
	}
	if remoteImg != nil && conf.Server.EnableM3UExternalAlbumArt {
		sf := func() (io.ReadCloser, string, error) { return fetchPlaylistImageURL(ctx, remoteImg) }
		if res, ok, isErr := resolveExternalStep(gate, "m3u", sf); ok {
			return res, nil
		} else if isErr {
			extErr = true
		}
	}

	albumIDs, err := ds.Playlist(ctx).Tracks(pl.ID, false).GetAlbumIDs(model.QueryOptions{Max: 4, Sort: "random()"})
	if err != nil {
		return resolution{}, err
	}

	var tiles []image.Image
	var tileErr error // first internal (non-external) tile failure, e.g. album deleted mid-flight
	for _, albumID := range albumIDs {
		res, err := resolveAlbum(ctx, ds, ag, ffm, albumID, gate, false)
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

// resolveMediaFile resolves a track's own embedded art only; there is no folder or
// external fallback, so disabled/missing cover art is a definitive absent.
func resolveMediaFile(ctx context.Context, ds model.DataStore, ffm ffmpeg.FFmpeg, id string) (resolution, error) {
	mf, err := ds.MediaFile(ctx).Get(id)
	if err != nil {
		return resolution{}, err
	}
	if !conf.Server.EnableMediaFileCoverArt || !mf.HasCoverArt {
		return resolution{}, nil
	}
	lib, err := loadLibraryView(ctx, ds, mf.LibraryID)
	if err != nil {
		return resolution{}, err
	}
	res, _ := resolveEmbedded(ctx, lib, ffm, mf.Path)
	return res, nil
}

// resolveExternalStep runs a single external sourceFunc through the named gate; used by
// the playlist ExternalImageURL step. ok reports a hit; extErr reports a non-not-found
// error (a not-found is a definitive "no", not a failure).
func resolveExternalStep(gate gateFunc, name string, sf sourceFunc) (res resolution, ok bool, extErr bool) {
	r, path, err := gate(name, sf)
	if r != nil {
		return resolution{reader: r, source: "external", sourcePath: path}, true, false
	}
	return resolution{}, false, err != nil && !errors.Is(err, model.ErrNotFound)
}

// classifyPlaylistImage splits a playlist ExternalImageURL into a local filesystem path
// (served file-backed) or a remote http(s) URL (fetched and stored); at most one is set.
func classifyPlaylistImage(imageURL string) (localPath string, remote *url.URL) {
	if imageURL == "" {
		return "", nil
	}
	u, err := url.Parse(imageURL)
	if err != nil {
		return imageURL, nil // unparseable → treat as a local path
	}
	switch u.Scheme {
	case "http", "https":
		return "", u
	case "file":
		return u.Path, nil
	default:
		return imageURL, nil
	}
}

// Like sources.go's fromURL but maps 404/410 to ErrNotFound (definitive), so a stale M3U
// cover URL falls through to the grid instead of retrying forever and tripping the breaker.
func fetchPlaylistImageURL(ctx context.Context, imageURL *url.URL) (io.ReadCloser, string, error) {
	hc := http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, imageURL.String(), nil)
	req.Header.Set("User-Agent", consts.HTTPUserAgent)
	resp, err := hc.Do(req) //nolint:gosec
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		resp.Body.Close()
		return nil, "", model.ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, "", fmt.Errorf("error retrieving artwork from %s: %s", imageURL, resp.Status)
	}
	return resp.Body, imageURL.String(), nil
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
// decodeTile runs on every sampled album's resolved bytes before processItem's
// own maxImageBytes/maxImagePixels guards apply, so it enforces them itself too.
func decodeTile(r io.ReadCloser) (image.Image, error) {
	data, err := readCapped(r)
	if err != nil {
		return nil, err
	}
	img, _, err := decodeCapped(data)
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
