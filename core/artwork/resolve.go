package artwork

import (
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"io/fs"
	"net/url"
	"os"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/model"
)

// resolution is one attempted acquisition outcome for an entity.
type resolution struct {
	reader     io.ReadCloser // nil when no source yielded an image
	source     string        // model.ItemArtwork.Source value: "folder", "embedded", "external", "upload", "generated"
	sourcePath string        // backing library/upload file (folder/upload: the image; embedded: the audio file); "" otherwise
	refMtime   int64         // sourcePath mtime (unix-nanoseconds) at resolution; 0 when no sourcePath
	// external source errored/timed out. With no reader it forces failed (never absent);
	// on a hit a higher-priority external step failed—serve this, but retry later.
	extError bool
	// a local source that should have been readable wasn't. With no reader it forces failed,
	// so a transient I/O fault never records absent.
	localError bool
}

// chainState carries what a priority walk has seen so far. A hit takes extErr with it so a
// transient external failure still retries; localErr is dropped, as the scanner re-lists changes.
type chainState struct{ extErr, localErr bool }

// try stamps the accumulated external failure onto a hit, and records the miss otherwise.
func (c *chainState) try(res resolution, ok bool) (resolution, bool) {
	if ok {
		res.extError = c.extErr
		return res, true
	}
	c.localErr = c.localErr || res.localError
	return resolution{}, false
}

// exhausted is the outcome when no source in the chain yielded an image.
func (c *chainState) exhausted() resolution {
	return resolution{extError: c.extErr, localError: c.localErr}
}

// externalSource holds the agents to ask and the rate limiter/circuit breaker to ask them through.
type externalSource struct {
	agents *agents.Agents
	gate   gateFunc
}

// resolver walks a kind's priority chain and returns the first hit; a nil ext means local-only.
type resolver struct {
	ds     model.DataStore
	ffmpeg ffmpeg.FFmpeg
	ext    *externalSource
}

func newResolver(ds model.DataStore, ag *agents.Agents, ffm ffmpeg.FFmpeg, gate gateFunc) *resolver {
	if gate == nil {
		gate = passthroughGate
	}
	return &resolver{ds: ds, ffmpeg: ffm, ext: &externalSource{agents: ag, gate: gate}}
}

// newLocalResolver builds a resolver that can neither reach the network nor sample album art
// for the worker-built grid.
func newLocalResolver(ds model.DataStore, ffm ffmpeg.FFmpeg) *resolver {
	return &resolver{ds: ds, ffmpeg: ffm}
}

func (r *resolver) resolve(ctx context.Context, item model.ArtworkQueueItem) (resolution, error) {
	kind, _ := model.ParseKind(item.ItemKind)
	switch kind {
	case model.KindAlbumArtwork:
		return r.resolveAlbum(ctx, item.ItemID)
	case model.KindArtistArtwork:
		return r.resolveArtist(ctx, item.ItemID)
	case model.KindPlaylistArtwork:
		return r.resolvePlaylist(ctx, item.ItemID)
	case model.KindRadioArtwork:
		return r.resolveRadio(ctx, item.ItemID)
	case model.KindMediaFileArtwork:
		return r.resolveMediaFile(ctx, item.ItemID)
	default:
		return resolution{}, fmt.Errorf("artwork: kind %q is not resolvable by the worker", item.ItemKind)
	}
}

// fetchExternalAlbum and fetchExternalArtist are the only places resolution touches the network,
// so a local-only resolver is stopped here rather than at each point in the chain walk.
func (r *resolver) fetchExternalAlbum(ctx context.Context, al model.Album) (io.ReadCloser, string, bool) {
	if r.ext == nil {
		return nil, "", false
	}
	return fetchAlbumImage(ctx, r.ext.agents, r.ext.gate, al)
}

func (r *resolver) fetchExternalArtist(ctx context.Context, ar model.Artist) (io.ReadCloser, string, bool) {
	if r.ext == nil {
		return nil, "", false
	}
	return fetchArtistImage(ctx, r.ext.agents, r.ext.gate, ar)
}

// resolveAlbum walks conf.Server.CoverArtPriority over the folder, embedded and external sources.
func (r *resolver) resolveAlbum(ctx context.Context, albumID string) (resolution, error) {
	al, err := r.ds.Album(ctx).Get(albumID)
	if err != nil {
		return resolution{}, err
	}
	_, imgFiles, _, err := loadAlbumFoldersPaths(ctx, r.ds, *al)
	if err != nil {
		return resolution{}, err
	}
	lib, err := loadLibraryView(ctx, r.ds, al.LibraryID)
	if err != nil {
		return resolution{}, err
	}

	var chain chainState
	for pattern := range strings.SplitSeq(strings.ToLower(conf.Server.CoverArtPriority), ",") {
		pattern = strings.TrimSpace(pattern)
		switch {
		case pattern == "embedded":
			if res, ok := chain.try(resolveEmbedded(ctx, lib, r.ffmpeg, al.EmbedArtPath)); ok {
				return res, nil
			}
		case pattern == "external":
			if rd, name, isErr := r.fetchExternalAlbum(ctx, *al); rd != nil {
				return resolution{reader: rd, source: "external:" + name}, nil
			} else if isErr {
				chain.extErr = true
			}
		case len(imgFiles) > 0:
			if res, ok := chain.try(resolveFolderFile(ctx, lib, imgFiles, pattern)); ok {
				return res, nil
			}
		}
	}
	return chain.exhausted(), nil
}

// resolveArtist tries the uploaded image first, then walks conf.Server.ArtistArtPriority.
func (r *resolver) resolveArtist(ctx context.Context, artistID string) (resolution, error) {
	ar, err := r.ds.Artist(ctx).Get(artistID)
	if err != nil {
		return resolution{}, err
	}
	upload, ok := resolveLocalFile(ar.UploadedImagePath(), "upload")
	if ok {
		return upload, nil
	}
	if upload.localError {
		// The upload outranks every other source; falling through would persist a lower-priority
		// image as if the upload were gone.
		return upload, nil
	}

	// Only consider albums where the artist is the sole album artist.
	als, err := r.ds.Album(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.And{
			squirrel.Eq{"album_artist_id": artistID},
			squirrel.Eq{"json_array_length(participants, '$.albumartist')": 1},
		},
	})
	if err != nil {
		return resolution{}, err
	}
	albumPaths, imgFiles, _, err := loadAlbumFoldersPaths(ctx, r.ds, als...)
	if err != nil {
		return resolution{}, err
	}
	artistFolder, _, err := loadArtistFolder(ctx, r.ds, als, albumPaths)
	if err != nil {
		return resolution{}, err
	}
	var lib libraryView
	if len(als) > 0 {
		lib, err = loadLibraryView(ctx, r.ds, als[0].LibraryID)
		if err != nil {
			return resolution{}, err
		}
	}

	var chain chainState
	for pattern := range strings.SplitSeq(strings.ToLower(conf.Server.ArtistArtPriority), ",") {
		pattern = strings.TrimSpace(pattern)
		switch {
		case pattern == "external":
			if rd, name, isErr := r.fetchExternalArtist(ctx, *ar); rd != nil {
				return resolution{reader: rd, source: "external:" + name}, nil
			} else if isErr {
				chain.extErr = true
			}
		case pattern == "image-folder":
			if res, ok := chain.try(resolveArtistImageFolder(ar)); ok {
				return res, nil
			}
		case strings.HasPrefix(pattern, "album/"):
			if lib.FS == nil {
				continue
			}
			if res, ok := chain.try(resolveFolderFile(ctx, lib, imgFiles, strings.TrimPrefix(pattern, "album/"))); ok {
				return res, nil
			}
		default:
			if lib.FS == nil || artistFolder == "" {
				continue
			}
			if res, ok := chain.try(resolveArtistFolderPattern(ctx, lib, artistFolder, pattern)); ok {
				return res, nil
			}
		}
	}
	return chain.exhausted(), nil
}

// resolvePlaylist tries the uploaded image, the sidecar and ExternalImageURL, then a generated grid.
func (r *resolver) resolvePlaylist(ctx context.Context, playlistID string) (resolution, error) {
	pl, err := r.ds.Playlist(ctx).Get(playlistID)
	if err != nil {
		return resolution{}, err
	}

	var extErr bool
	for _, src := range []struct{ path, source string }{
		{pl.UploadedImagePath(), "upload"},
		{findPlaylistSidecarPath(ctx, pl.Path), "folder"},
	} {
		res, ok := resolveLocalFile(src.path, src.source)
		if ok {
			return res, nil
		}
		if res.localError {
			// These outrank the generated grid; falling through would replace them with it.
			return res, nil
		}
	}
	// A local ExternalImageURL is file-backed and served in place; only http(s) needs the gated fetch.
	localImg, remoteImg := classifyPlaylistImage(pl.ExternalImageURL)
	if localImg != "" {
		res, ok := resolveLocalFile(localImg, "folder")
		if ok {
			return res, nil
		}
		if res.localError {
			return res, nil
		}
	}
	if r.ext == nil {
		// The remote fetch and the generated grid are worker-only; a request must do neither.
		return resolution{}, nil
	}
	if remoteImg != nil && conf.Server.EnableM3UExternalAlbumArt {
		sf := func() (io.ReadCloser, string, error) { return fromURL(ctx, remoteImg) }
		if res, ok, isErr := resolveExternalStep(r.ext.gate, "m3u", sf); ok {
			return res, nil
		} else if isErr {
			extErr = true
		}
	}

	albumIDs, err := r.ds.Playlist(ctx).Tracks(pl.ID, false).GetAlbumIDs(model.QueryOptions{Max: 4, Sort: "random()"})
	if err != nil {
		return resolution{}, err
	}

	var tiles []image.Image
	var tileErr error // first internal (non-external) tile failure, e.g. album deleted mid-flight
	for _, albumID := range albumIDs {
		res, err := r.resolveAlbum(ctx, albumID)
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
		// A tile-level failure must never resolve as a clean absent.
		if tileErr != nil {
			return resolution{}, fmt.Errorf("resolvePlaylist: sampled album art failed: %w", tileErr)
		}
		return resolution{extError: extErr}, nil
	}
	// Grow to 4 tiles by repeating what we have.
	switch len(tiles) {
	case 2:
		tiles = append(tiles, tiles[1], tiles[0])
	case 3:
		tiles = append(tiles, tiles[0])
	}
	grid, err := assembleTiles(tiles)
	if err != nil {
		return resolution{extError: extErr}, nil //nolint:nilerr // encode failure is a soft "no image", not a resolution error
	}
	return resolution{reader: grid, source: "generated", extError: extErr}, nil
}

// resolveRadio serves only an uploaded image; there is no fallback.
func (r *resolver) resolveRadio(ctx context.Context, radioID string) (resolution, error) {
	radio, err := r.ds.Radio(ctx).Get(radioID)
	if err != nil {
		return resolution{}, err
	}
	res, _ := resolveLocalFile(radio.UploadedImagePath(), "upload")
	return res, nil
}

// resolveMediaFile resolves a track's own embedded art only, so disabled or missing cover art
// is a definitive absent.
func (r *resolver) resolveMediaFile(ctx context.Context, id string) (resolution, error) {
	mf, err := r.ds.MediaFile(ctx).Get(id)
	if err != nil {
		return resolution{}, err
	}
	if !conf.Server.EnableMediaFileCoverArt || !mf.HasCoverArt {
		return resolution{}, nil
	}
	lib, err := loadLibraryView(ctx, r.ds, mf.LibraryID)
	if err != nil {
		return resolution{}, err
	}
	res, _ := resolveEmbedded(ctx, lib, r.ffmpeg, mf.Path)
	return res, nil
}

// resolveExternalStep runs a single external sourceFunc through the named gate. extErr excludes
// a not-found, which is a definitive "no" rather than a failure.
func resolveExternalStep(gate gateFunc, name string, sf sourceFunc) (res resolution, ok bool, extErr bool) {
	r, path, err := gate(name, sf)
	if r != nil {
		return resolution{reader: r, source: "external", sourcePath: path}, true, false
	}
	return resolution{}, false, err != nil && !errors.Is(err, model.ErrNotFound)
}

// classifyPlaylistImage splits a playlist ExternalImageURL into a local filesystem path or a
// remote http(s) URL; at most one is set.
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

func resolveEmbedded(ctx context.Context, lib libraryView, ffm ffmpeg.FFmpeg, embedRel string) (resolution, bool) {
	if embedRel == "" {
		return resolution{}, false
	}
	abs := lib.Abs(embedRel)
	var unreadable bool
	for _, sf := range []sourceFunc{fromTag(ctx, lib.FS, embedRel), fromFFmpegTag(ctx, ffm, abs)} {
		r, _, err := sf()
		if r != nil {
			return resolution{reader: r, source: "embedded", sourcePath: abs, refMtime: mtimeViaFS(lib.FS, embedRel)}, true
		}
		unreadable = unreadable || errors.Is(err, errSourceUnreadable)
	}
	return resolution{localError: unreadable}, false
}

func resolveFolderFile(ctx context.Context, lib libraryView, imgFiles []string, pattern string) (resolution, bool) {
	r, path, err := fromExternalFile(ctx, lib.FS, imgFiles, pattern)()
	if r == nil {
		return resolution{localError: errors.Is(err, errSourceUnreadable)}, false
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
	r, path, err := fromArtistFolder(ctx, lib.FS, lib.absRoot, artistFolder, pattern)()
	if r == nil {
		return resolution{localError: errors.Is(err, errSourceUnreadable)}, false
	}
	return resolution{reader: r, source: "folder", sourcePath: path, refMtime: mtimeOf(path)}, true
}

// resolveLocalFile opens an absolute path directly. A missing path is "no source"; any other
// open failure says nothing about whether the image exists.
func resolveLocalFile(path, source string) (resolution, bool) {
	if path == "" {
		return resolution{}, false
	}
	f, err := os.Open(path)
	if err != nil {
		return resolution{localError: !errors.Is(err, fs.ErrNotExist)}, false
	}
	return resolution{reader: f, source: source, sourcePath: path, refMtime: mtimeOf(path)}, true
}

func mtimeOf(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.ModTime().UnixNano()
}

// mtimeViaFS stats through the library FS, since library roots in tests may not be real OS paths.
func mtimeViaFS(fsys fs.FS, name string) int64 {
	if fsys == nil || name == "" {
		return 0
	}
	info, err := fs.Stat(fsys, name)
	if err != nil {
		return 0
	}
	return info.ModTime().UnixNano()
}
