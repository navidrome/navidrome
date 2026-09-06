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
	"path"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
)

// resolution is one attempted acquisition outcome for an entity.
type resolution struct {
	reader     io.ReadCloser // nil when no source yielded an image
	source     string        // model.ItemArtwork.Source value: "folder", "embedded", "external", "upload", "generated"
	sourcePath string        // backing library/upload file (folder/upload: the image; embedded: the audio file); "" otherwise
	refMtime   int64         // sourcePath mtime (unix-nanoseconds) at resolution; 0 when no sourcePath
	// a faulted external source, carrying the provider's requested delay when it named one.
	// With no reader it forces failed (never absent); on a hit, serve this but retry later.
	extErr error
	// a local source that should have been readable wasn't. With no reader it forces failed,
	// so a transient I/O fault never records absent.
	localError bool
}

// chainState carries what a priority walk has seen so far. A hit takes extErr with it so a
// transient external failure still retries; localErr is dropped, as the scanner re-lists changes.
type chainState struct {
	extErr   error
	localErr bool
	trace    *ChainTrace // nil only where no caller attached one
}

// try stamps the accumulated external failure onto a hit, and records the miss otherwise.
func (c *chainState) try(candidate string, res resolution, ok bool) (resolution, bool) {
	if ok {
		res.extErr = c.extErr
		c.record(candidate, OutcomeHit, res.sourcePath)
		return res, true
	}
	c.localErr = c.localErr || res.localError
	if res.localError {
		c.record(candidate, OutcomeUnreadable, "")
	} else {
		c.record(candidate, OutcomeMiss, "")
	}
	return resolution{}, false
}

func (c *chainState) record(candidate string, out Outcome, detail string) {
	c.trace.add(TraceStep{Candidate: candidate, Outcome: out, Detail: detail})
}

// exhausted is the outcome when no source in the chain yielded an image.
func (c *chainState) exhausted() resolution {
	return resolution{extErr: c.extErr, localError: c.localErr}
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

// Explainable reports whether TracingResolver can walk this kind's sources and report which one
// won; playlists and radios resolve from a fixed internal order, with nothing configured to explain.
func Explainable(kind model.Kind) bool {
	switch kind {
	case model.KindArtistArtwork, model.KindAlbumArtwork, model.KindDiscArtwork, model.KindMediaFileArtwork:
		return true
	}
	return false
}

// MayFetchExternal reports whether resolving this kind can issue an external request under the
// current config. Playlists inherit the album chain: the generated grid resolves album art.
func MayFetchExternal(kind model.Kind) bool {
	switch kind {
	case model.KindArtistArtwork:
		return chainFetchesExternal(conf.Server.ArtistArtPriority)
	case model.KindAlbumArtwork:
		return chainFetchesExternal(conf.Server.CoverArtPriority)
	case model.KindPlaylistArtwork:
		return conf.Server.EnableM3UExternalAlbumArt || chainFetchesExternal(conf.Server.CoverArtPriority)
	default:
		return false
	}
}

// ImageAgentCount is how many enabled agents provide artist and album images.
type ImageAgentCount struct{ Artist, Album int }

// NewImageAgentCount counts what an external step would consult, so an estimate and the gate that
// guards it cannot disagree about which agents exist.
func NewImageAgentCount(ag *agents.Agents) ImageAgentCount {
	if ag == nil {
		return ImageAgentCount{}
	}
	return ImageAgentCount{Artist: len(ag.ArtistImageAgents()), Album: len(ag.AlbumImageAgents())}
}

// ExternalLookupsPerItem reports what resolving one item of this kind can cost: every image agent is
// tried, and a zero count still bills one, so agents the caller cannot see never read as free.
func ExternalLookupsPerItem(kind model.Kind, agents ImageAgentCount) int64 {
	if !MayFetchExternal(kind) {
		return 0
	}
	switch kind {
	case model.KindArtistArtwork:
		return int64(max(agents.Artist, 1))
	case model.KindAlbumArtwork:
		return int64(max(agents.Album, 1))
	case model.KindPlaylistArtwork:
		var n int64
		if conf.Server.EnableM3UExternalAlbumArt {
			n++
		}
		if chainFetchesExternal(conf.Server.CoverArtPriority) {
			n += PlaylistGridSamples * int64(max(agents.Album, 1))
		}
		return n
	}
	return 0
}

func chainFetchesExternal(priority string) bool {
	for pattern := range strings.SplitSeq(strings.ToLower(priority), ",") {
		if strings.TrimSpace(pattern) == externalCandidate {
			return true
		}
	}
	return false
}

// Album and artist fetches stop here when the resolver is local-only, rather than at each point in
// the chain walk; resolvePlaylist gates the third network path, the m3u image URL, itself.
func (r *resolver) fetchExternalAlbum(ctx context.Context, al model.Album) (io.ReadCloser, string, error) {
	if r.ext == nil {
		return nil, "", nil
	}
	return fetchAlbumImage(ctx, r.ext.agents, r.ext.gate, al)
}

func (r *resolver) fetchExternalArtist(ctx context.Context, ar model.Artist) (io.ReadCloser, string, error) {
	if r.ext == nil {
		return nil, "", nil
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

	chain := chainState{trace: traceFrom(ctx)}
	for pattern := range strings.SplitSeq(strings.ToLower(conf.Server.CoverArtPriority), ",") {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		switch {
		case pattern == "embedded":
			res, ok := resolveEmbedded(ctx, lib, r.ffmpeg, al.EmbedArtPath)
			if res, ok = chain.try(pattern, res, ok); ok {
				return res, nil
			}
		case pattern == externalCandidate:
			if rd, name, err := r.fetchExternalAlbum(ctx, *al); rd != nil {
				return resolution{reader: rd, source: ExternalPrefix + name}, nil
			} else if err != nil {
				chain.extErr = longerRetry(chain.extErr, err)
			}
		case len(imgFiles) > 0:
			res, ok := resolveFolderFile(ctx, lib, imgFiles, pattern)
			if res, ok = chain.try(pattern, res, ok); ok {
				return res, nil
			}
		default:
			chain.record(pattern, OutcomeSkipped, "no images in album folder")
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
	chain := chainState{trace: traceFrom(ctx)}
	upload, uploadOK := resolveLocalFile(ar.UploadedImagePath(), "upload")
	if res, ok := chain.try("upload", upload, uploadOK); ok {
		return res, nil
	}
	if upload.localError {
		// The upload outranks every other source; falling through would persist a lower-priority
		// image as if the upload were gone.
		return upload, nil
	}

	// Only consider albums where the artist is the sole album artist.
	als, err := r.ds.Album(ctx).GetAll(model.QueryOptions{Filters: persistence.SoleAlbumArtistFilter(artistID)})
	if err != nil {
		return resolution{}, err
	}
	albumPaths, imgFiles, _, err := loadArtistAlbumRoots(ctx, r.ds, als)
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

	for pattern := range strings.SplitSeq(strings.ToLower(conf.Server.ArtistArtPriority), ",") {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		switch {
		case pattern == externalCandidate:
			if rd, name, err := r.fetchExternalArtist(ctx, *ar); rd != nil {
				return resolution{reader: rd, source: ExternalPrefix + name}, nil
			} else if err != nil {
				chain.extErr = longerRetry(chain.extErr, err)
			}
		case pattern == "image-folder":
			res, ok := resolveArtistImageFolder(ar)
			if res, ok = chain.try(pattern, res, ok); ok {
				return res, nil
			}
		case strings.HasPrefix(pattern, "album/"):
			if lib.FS == nil {
				chain.record(pattern, OutcomeSkipped, "artist has no albums")
				continue
			}
			res, ok := resolveFolderFile(ctx, lib, imgFiles, strings.TrimPrefix(pattern, "album/"))
			if res, ok = chain.try(pattern, res, ok); ok {
				return res, nil
			}
		default:
			if lib.FS == nil {
				chain.record(pattern, OutcomeSkipped, "artist has no albums")
				continue
			}
			if artistFolder == "" {
				chain.record(pattern, OutcomeSkipped, "no artist folder")
				continue
			}
			res, ok := resolveArtistFolderPattern(ctx, lib, artistFolder, pattern)
			if res, ok = chain.try(pattern, res, ok); ok {
				return res, nil
			}
		}
	}
	return chain.exhausted(), nil
}

// PlaylistGridSamples is how many albums resolvePlaylist samples to build the generated grid.
const PlaylistGridSamples = 4

// resolvePlaylist tries the uploaded image, the sidecar and ExternalImageURL, then a generated grid.
func (r *resolver) resolvePlaylist(ctx context.Context, playlistID string) (resolution, error) {
	pl, err := r.ds.Playlist(ctx).Get(playlistID)
	if err != nil {
		return resolution{}, err
	}

	var extErr error
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
		if res, ok, err := resolveExternalStep(r.ext.gate, "m3u", sf); ok {
			return res, nil
		} else if err != nil {
			extErr = longerRetry(extErr, err)
			// Record it here with its detail: once album sampling adds its own steps, the processor's
			// empty-trace fallback no longer fires, and the error that forced the retry would be lost.
			traceFrom(ctx).add(TraceStep{Candidate: ExternalPrefix + "m3u", Outcome: OutcomeError, Detail: err.Error()})
		}
	}

	albumIDs, err := r.ds.Playlist(ctx).Tracks(pl.ID, false).
		GetAlbumIDs(model.QueryOptions{Max: PlaylistGridSamples, Sort: "random()"})
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
		if res.extErr != nil {
			extErr = longerRetry(extErr, res.extErr)
		}
		if res.reader == nil {
			continue
		}
		tile, decErr := decodeTile(res.reader)
		res.reader.Close()
		if decErr == nil {
			tiles = append(tiles, tile)
		}
		if len(tiles) == PlaylistGridSamples {
			break
		}
	}
	if len(tiles) == 0 {
		// A tile-level failure must never resolve as a clean absent.
		if tileErr != nil {
			return resolution{}, fmt.Errorf("resolvePlaylist: sampled album art failed: %w", tileErr)
		}
		return resolution{extErr: extErr}, nil
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
		return resolution{extErr: extErr}, nil //nolint:nilerr // encode failure is a soft "no image", not a resolution error
	}
	return resolution{reader: grid, source: "generated", extErr: extErr}, nil
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
	chain := chainState{trace: traceFrom(ctx)}
	switch {
	case !conf.Server.EnableMediaFileCoverArt:
		chain.record("embedded", OutcomeSkipped, "EnableMediaFileCoverArt is off")
		return resolution{}, nil
	case !mf.HasCoverArt:
		chain.record("embedded", OutcomeMiss, "the track has no embedded cover art")
		return resolution{}, nil
	}
	lib, err := loadLibraryView(ctx, r.ds, mf.LibraryID)
	if err != nil {
		return resolution{}, err
	}
	res, ok := resolveEmbedded(ctx, lib, r.ffmpeg, mf.Path)
	if res, ok = chain.try("embedded", res, ok); ok {
		return res, nil
	}
	return chain.exhausted(), nil
}

// resolveDisc walks conf.Server.DiscArtPriority. Disc artwork keeps no state row and is never
// queued: the serving path reads it through on every request, so this only ever explains.
func (r *resolver) resolveDisc(ctx context.Context, id string) (resolution, error) {
	dr, err := newDiscArtworkReader(ctx, r.ds, model.ArtworkID{Kind: model.KindDiscArtwork, ID: id})
	if err != nil {
		return resolution{}, err
	}
	chain := chainState{trace: traceFrom(ctx)}
	return dr.selectImage(ctx, r.ffmpeg, conf.Server.DiscArtPriority, &chain)
}

// resolveExternalStep runs a single external sourceFunc through the named gate. A not-found is a
// definitive "no", returned as (_, false, nil); any other error is a failure the caller records.
func resolveExternalStep(gate gateFunc, name string, sf sourceFunc) (resolution, bool, error) {
	r, path, err := gate(name, sf)
	if r != nil {
		return resolution{reader: r, source: externalCandidate, sourcePath: path}, true, nil
	}
	if errors.Is(err, model.ErrNotFound) {
		return resolution{}, false, nil
	}
	return resolution{}, false, err
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

// resolveFolderSource turns a source that yields a library-relative image path into a folder
// resolution, keeping an existing-but-unopenable file distinct from an absent one.
func resolveFolderSource(lib libraryView, sf sourceFunc) (resolution, bool) {
	r, path, err := sf()
	if r == nil {
		return resolution{localError: errors.Is(err, errSourceUnreadable)}, false
	}
	return resolution{reader: r, source: "folder", sourcePath: lib.Abs(path), refMtime: mtimeViaFS(lib.FS, path)}, true
}

func resolveFolderFile(ctx context.Context, lib libraryView, imgFiles []string, pattern string) (resolution, bool) {
	return resolveFolderSource(lib, fromExternalFile(ctx, lib.FS, imgFiles, pattern))
}

// IsArtistImageFile reports whether a file name matches any file-glob token of ArtistArtPriority.
// Basename-only on purpose: the chain climbs parent folders, so a token's prefix is not fixed.
func IsArtistImageFile(name string) bool {
	name = strings.ToLower(name)
	for pattern := range strings.SplitSeq(strings.ToLower(conf.Server.ArtistArtPriority), ",") {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" || pattern == externalCandidate || pattern == "image-folder" {
			continue
		}
		if ok, _ := path.Match(path.Base(pattern), name); ok {
			return true
		}
	}
	return false
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
		// Carry the source label even on a fault, so a resolver with no chain (playlist/radio) can
		// still name what faulted in the trace.
		return resolution{source: source, localError: !errors.Is(err, fs.ErrNotExist)}, false
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
