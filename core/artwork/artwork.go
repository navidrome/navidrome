package artwork

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/resources"
	"github.com/navidrome/navidrome/utils/cache"
)

var ErrUnavailable = errors.New("artwork unavailable")

// errStaleSource means the backing file's mtime no longer matches RefMtime, so the stored hash may be stale.
var errStaleSource = errors.New("artwork: source file changed since resolution")

// Image is one servable artwork response.
type Image struct {
	io.ReadCloser
	Hash        string // pixel identity; "" for placeholders
	ETag        string // representation validator; "" means Hash applies (full-size original)
	LastUpdated time.Time
	Placeholder bool
}

// representationTag varies with dimensions and encode settings, so a config change invalidates
// a revalidating client's cache even though the pixel hash is unchanged.
func representationTag(hash string, size int, square bool) string {
	return fmt.Sprintf("%s.%d.%v.%s", hash, size, square, formatQualityTag())
}

type Artwork interface {
	// Get returns ErrUnavailable when there is nothing to serve and model.ErrNotFound when
	// the id resolves to nothing, so the caller can pick placeholder vs 404.
	Get(ctx context.Context, artID model.ArtworkID, size int, square bool) (*Image, error)
	// GetOrPlaceholder accepts an artwork token or a raw entity id, falling back to the
	// kind's placeholder image (never resized, Placeholder=true).
	GetOrPlaceholder(ctx context.Context, id string, size int, square bool) (*Image, error)
}

func NewArtwork(ds model.DataStore, cache cache.FileCache, store *ImageStore, ffm ffmpeg.FFmpeg) Artwork {
	return &service{ds: ds, cache: cache, store: store, ffmpeg: ffm}
}

// entityExists reports whether the entity an artwork id points at is still there: state rows
// outlive a deleted entity until the next prune, so a servable row is not evidence of its owner.
func entityExists(ctx context.Context, ds model.DataStore, artID model.ArtworkID) bool {
	var found bool
	var err error
	switch artID.Kind {
	case model.KindArtistArtwork:
		found, err = ds.Artist(ctx).Exists(artID.ID)
	case model.KindAlbumArtwork:
		found, err = ds.Album(ctx).Exists(artID.ID)
	case model.KindMediaFileArtwork:
		found, err = ds.MediaFile(ctx).Exists(artID.ID)
	case model.KindPlaylistArtwork:
		found, err = ds.Playlist(ctx).Exists(artID.ID)
	case model.KindRadioArtwork:
		found, err = ds.Radio(ctx).Exists(artID.ID)
	case model.KindDiscArtwork:
		albumID, _, perr := model.ParseDiscArtworkID(artID.ID)
		if perr != nil {
			return false
		}
		found, err = ds.Album(ctx).Exists(albumID)
	default:
		return false
	}
	return err == nil && found
}

type service struct {
	ds     model.DataStore
	cache  cache.FileCache
	store  *ImageStore
	ffmpeg ffmpeg.FFmpeg
}

func (s *service) GetOrPlaceholder(ctx context.Context, id string, size int, square bool) (*Image, error) {
	artID, err := s.parseArtworkID(ctx, id)
	var img *Image
	if err == nil {
		img, err = s.Get(ctx, artID, size, square)
	}
	// Only a resolvable entity with no art gets a placeholder; an unknown id must stay
	// ErrNotFound so callers can still answer 404 / Subsonic error 70.
	if errors.Is(err, ErrUnavailable) {
		return placeholderImage(artID.Kind), nil
	}
	return img, err
}

func (s *service) Get(ctx context.Context, artID model.ArtworkID, size int, square bool) (*Image, error) {
	if artID.ID == "" {
		return nil, ErrUnavailable
	}
	if size < 0 {
		size = 0 // a negative size means full-size, not a giant (OOM) resize rectangle
	}
	switch artID.Kind {
	case model.KindDiscArtwork:
		return s.serveDisc(ctx, artID, size, square)
	case model.KindMediaFileArtwork:
		return s.serveMediaFile(ctx, artID, size, square)
	default:
		return s.serveEntity(ctx, artID, size, square)
	}
}

// requestRecheckAge throttles view-triggered rechecks so reopening a genuinely-absent page can't
// hammer external services; below StaleAbsentAge to catch younger absences.
const requestRecheckAge = time.Hour

func (s *service) serveEntity(ctx context.Context, artID model.ArtworkID, size int, square bool) (*Image, error) {
	ia, err := s.ds.Artwork(ctx).GetItemArtwork(artID.Kind, artID.ID, model.ImageTypePrimary)
	switch {
	case errors.Is(err, model.ErrNotFound):
		return s.provisional(ctx, artID, size, square)
	case err != nil:
		return nil, err
	case ia.Hash == "":
		// Inserts an immediately-eligible recheck for a settled absent row.
		if time.Since(ia.AttemptedAt) > requestRecheckAge {
			s.enqueue(ctx, artID, model.ArtworkPriorityBump)
		}
		return nil, ErrUnavailable
	default:
		return s.serveHash(ctx, artID, ia, size, square)
	}
}

// serveSource is the one place bytes become an Image. hash is the pixel identity ("" for disc art)
// and doubles as the full-size validator, so an ETag is only needed when resized or hash is "".
func (s *service) serveSource(ctx context.Context, key, hash string, lastUpdate time.Time,
	size int, square bool, open func() (io.ReadCloser, error),
) (*Image, error) {
	if size == 0 && !square {
		rc, err := open()
		if err != nil {
			return nil, err
		}
		if rc == nil {
			return nil, ErrUnavailable
		}
		img := &Image{ReadCloser: rc, Hash: hash, LastUpdated: lastUpdate}
		if hash == "" {
			img.ETag = representationTag(key, size, square)
		}
		return img, nil
	}
	stream, err := s.cache.Get(ctx, &resizedItem{
		hash: key, size: size, square: square, ffmpeg: s.ffmpeg, open: open,
	})
	if err != nil {
		return nil, err
	}
	return &Image{ReadCloser: stream, Hash: hash, ETag: representationTag(key, size, square), LastUpdated: lastUpdate}, nil
}

// serveHash serves the bytes of a found state row. A mismatch/open error is dangling, but a
// cancelled request is not: it must not enqueue a re-resolution.
func (s *service) serveHash(ctx context.Context, artID model.ArtworkID, ia *model.ItemArtwork, size int, square bool) (*Image, error) {
	// Only this path can hand back a deleted entity's bytes; the others load their entity anyway.
	if !entityExists(ctx, s.ds, artID) {
		return nil, ErrUnavailable
	}
	art, err := s.ds.Artwork(ctx).GetImage(ia.Hash)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return s.dangling(ctx, artID)
		}
		return nil, err
	}
	img, err := s.serveSource(ctx, ia.Hash, ia.Hash, ia.UpdatedAt, size, square,
		func() (io.ReadCloser, error) { return openOriginal(ia, art.Mime, s.store) })
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		log.Warn(ctx, "Artwork: Could not serve image", "artID", artID, "size", size, err)
		return s.dangling(ctx, artID)
	}
	return img, nil
}

// openOriginal enforces the mtime invariant: bytes are never served under a hash they no longer match.
func openOriginal(ia *model.ItemArtwork, mime string, store *ImageStore) (io.ReadCloser, error) {
	if isFileBacked(ia.Source) {
		f, err := os.Open(ia.SourcePath)
		if err != nil {
			return nil, err
		}
		info, err := f.Stat()
		if err != nil {
			f.Close()
			return nil, err
		}
		if ia.RefMtime != 0 && info.ModTime().UnixNano() != ia.RefMtime {
			f.Close()
			log.Debug("Artwork: Backing file changed since resolution", "path", ia.SourcePath,
				"hash", ia.Hash, "resolvedMtime", ia.RefMtime, "currentMtime", info.ModTime().UnixNano())
			return nil, errStaleSource
		}
		return f, nil
	}
	// Store-backed bytes still carry the source's mtime, to detect edits to embedded art.
	if ia.SourcePath != "" && ia.RefMtime != 0 {
		info, err := os.Stat(ia.SourcePath)
		if err != nil {
			return nil, err
		}
		if info.ModTime().UnixNano() != ia.RefMtime {
			log.Debug("Artwork: Source file changed since resolution", "path", ia.SourcePath,
				"hash", ia.Hash, "resolvedMtime", ia.RefMtime, "currentMtime", info.ModTime().UnixNano())
			return nil, errStaleSource
		}
	}
	return store.Open(ia.Hash, mime)
}

// provisional serves local bytes for an entity with no state row, enqueuing the worker but
// never writing a state row itself.
func (s *service) provisional(ctx context.Context, artID model.ArtworkID, size int, square bool) (*Image, error) {
	item := model.ArtworkQueueItem{ItemKind: artID.Kind.Prefix(), ItemID: artID.ID, ImageType: model.ImageTypePrimary}
	res, err := newLocalResolver(s.ds, s.ffmpeg).resolve(ctx, item)
	if err != nil {
		return nil, err
	}
	s.enqueue(ctx, artID, model.ArtworkPriorityBump)
	log.Debug(ctx, "Artwork: Provisional read-through, no state row yet", "artID", artID,
		"source", res.source, "hit", res.reader != nil)
	return s.serveResolution(ctx, res, size, square)
}

// serveResolution turns a local resolution's bytes into a servable Image (byte-hash only, no decode).
func (s *service) serveResolution(ctx context.Context, res resolution, size int, square bool) (*Image, error) {
	if res.reader == nil {
		return nil, ErrUnavailable
	}
	defer res.reader.Close()
	data, err := readCapped(res.reader)
	if err != nil {
		return nil, ErrUnavailable
	}
	hash, err := hashImage(bytes.NewReader(data))
	if err != nil {
		return nil, ErrUnavailable
	}
	// Keyed by the byte-hash, so the entry lines up with the worker's eventual store entry.
	return s.serveSource(ctx, hash, hash, unixMtime(res.refMtime), size, square,
		func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(data)), nil })
}

func (s *service) serveMediaFile(ctx context.Context, artID model.ArtworkID, size int, square bool) (*Image, error) {
	// The setting is not in the config fingerprint, so honor it at serve time: a direct mf- URL
	// must fall back to disc/album instead of serving stale persisted embedded art.
	if !conf.Server.EnableMediaFileCoverArt {
		mf, err := s.ds.MediaFile(ctx).Get(artID.ID)
		if err != nil {
			return nil, err
		}
		return s.Get(ctx, mf.DiscCoverArtID(), size, square)
	}
	ia, err := s.ds.Artwork(ctx).GetItemArtwork(model.KindMediaFileArtwork, artID.ID, model.ImageTypePrimary)
	switch {
	case err == nil && ia.Hash != "":
		return s.serveHash(ctx, artID, ia, size, square)
	case err == nil:
		// absent row: fall through
	case errors.Is(err, model.ErrNotFound):
		// no row: fall through
	default:
		return nil, err
	}
	noRow := errors.Is(err, model.ErrNotFound)

	mf, err := s.ds.MediaFile(ctx).Get(artID.ID)
	if err != nil {
		return nil, err
	}
	if noRow && conf.Server.EnableMediaFileCoverArt && mf.HasCoverArt {
		return s.provisionalEmbedded(ctx, artID, *mf, size, square)
	}
	// Mirror MediaFile.CoverArtID: a track defers to its disc art, which falls back to the album.
	return s.Get(ctx, mf.DiscCoverArtID(), size, square)
}

// provisionalEmbedded serves a track's embedded art immediately, leaving the state row to the worker.
func (s *service) provisionalEmbedded(ctx context.Context, artID model.ArtworkID, mf model.MediaFile, size int, square bool) (*Image, error) {
	lib, err := loadLibraryView(ctx, s.ds, mf.LibraryID)
	if err != nil {
		return nil, err
	}
	res, ok := resolveEmbedded(ctx, lib, s.ffmpeg, mf.Path)
	s.enqueue(ctx, artID, model.ArtworkPriorityBump)
	if !ok {
		// Eligible but unextractable: fall back the way CoverArtID does, not to a placeholder.
		return s.Get(ctx, mf.DiscCoverArtID(), size, square)
	}
	return s.serveResolution(ctx, res, size, square)
}

// serveDisc reads disc art through with no state row and no enqueue, falling back to the album cover.
func (s *service) serveDisc(ctx context.Context, artID model.ArtworkID, size int, square bool) (*Image, error) {
	dr, err := newDiscArtworkReader(ctx, s.ds, artID)
	if err != nil {
		return nil, err
	}
	// Single-disc albums run the chain too: a disc can carry art distinct from the album cover.
	selectImage := func() (io.ReadCloser, string, error) {
		funcs := dr.fromDiscArtPriority(ctx, s.ffmpeg, conf.Server.DiscArtPriority)
		return selectImageReader(ctx, artID, funcs...)
	}
	albumArtID := model.ArtworkID{Kind: model.KindAlbumArtwork, ID: dr.album.ID}
	// Disc art has no state row, hence no content hash: keying on id, album mtime and
	// DiscArtPriority lets a warm cache answer without running the chain or touching the disk.
	key := fmt.Sprintf("%s|%d|%s", artID.ID, dr.cacheTime().UnixNano(), conf.Server.DiscArtPriority)
	img, err := s.serveSource(ctx, key, "", dr.cacheTime(), size, square,
		func() (io.ReadCloser, error) { rc, _, err := selectImage(); return rc, err })
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		return s.Get(ctx, albumArtID, size, square)
	}
	return img, nil
}

// dangling enqueues a re-resolution and reports unavailable, leaving the state row untouched.
func (s *service) dangling(ctx context.Context, artID model.ArtworkID) (*Image, error) {
	log.Debug(ctx, "Artwork: State row points at bytes we cannot serve, re-resolving", "artID", artID)
	s.enqueue(ctx, artID, model.ArtworkPriorityScan)
	return nil, ErrUnavailable
}

func (s *service) enqueue(ctx context.Context, artID model.ArtworkID, priority int) {
	err := s.ds.ArtworkQueue(ctx).EnqueuePreservingBackoff(model.ArtworkQueueItem{
		ItemKind:  artID.Kind.Prefix(),
		ItemID:    artID.ID,
		ImageType: model.ImageTypePrimary,
		Priority:  priority,
	})
	if err != nil {
		log.Warn(ctx, "Artwork: Could not enqueue re-resolution", "artID", artID, err)
	}
}

func placeholderImage(kind model.Kind) *Image {
	path := consts.PlaceholderAlbumArt
	if kind == model.KindArtistArtwork {
		path = consts.PlaceholderArtistArt
	}
	r, _ := resources.FS().Open(path)
	return &Image{ReadCloser: r, Placeholder: true}
}

type coverArtIDGetter interface {
	CoverArtID() model.ArtworkID
}

// parseArtworkID accepts an artwork token or a raw entity id, resolving the latter to its CoverArtID.
func (s *service) parseArtworkID(ctx context.Context, id string) (model.ArtworkID, error) {
	if id == "" {
		return model.ArtworkID{}, ErrUnavailable
	}
	if artID, err := model.ParseArtworkID(id); err == nil {
		return artID, nil
	}
	entity, err := model.GetEntityByID(ctx, s.ds, id)
	if err != nil {
		return model.ArtworkID{}, err
	}
	if e, ok := entity.(coverArtIDGetter); ok {
		return e.CoverArtID(), nil
	}
	return model.ArtworkID{}, model.ErrNotFound
}

// TracingResolver is the CLI's read-only view of resolution: it walks the priority chain, records
// the walk and reports the winning source, without ever writing artwork state.
type TracingResolver struct {
	inner *resolver
	trace *ChainTrace
}

// NewTracingResolver builds a TracingResolver that records its priority-chain walk. With live
// false the external tier is reported but never called.
func NewTracingResolver(ds model.DataStore, ag *agents.Agents, ffm ffmpeg.FFmpeg, t *ChainTrace, live bool) *TracingResolver {
	gate := offlineGate(t)
	if live {
		// A diagnostic must show the provider's real answer, and one item is at most one call
		// per agent, so --live deliberately bypasses the rate limiter and circuit breaker.
		gate = tracingGate(t, passthroughGate)
	}
	return &TracingResolver{inner: newResolver(ds, ag, ffm, gate), trace: t}
}

// Resolve walks kind's sources for id, recording the walk, and reports the winning source
// ("" when none produced an image).
func (r *TracingResolver) Resolve(ctx context.Context, kind model.Kind, id string) (string, error) {
	switch kind {
	case model.KindArtistArtwork:
		return r.explain(ctx, r.inner.resolveArtist, id)
	case model.KindAlbumArtwork:
		return r.explain(ctx, r.inner.resolveAlbum, id)
	case model.KindDiscArtwork:
		return r.explain(ctx, r.inner.resolveDisc, id)
	case model.KindMediaFileArtwork:
		return r.explain(ctx, r.inner.resolveMediaFile, id)
	}
	return "", fmt.Errorf("artwork: %s artwork has no chain to explain", kind)
}

// explain discards the bytes: nothing downstream persists this resolution, so nothing else
// would close the reader either.
func (r *TracingResolver) explain(ctx context.Context, resolve func(context.Context, string) (resolution, error), id string) (string, error) {
	res, err := resolve(withTrace(ctx, r.trace), id)
	if err != nil {
		return "", err
	}
	if res.reader != nil {
		_ = res.reader.Close()
	}
	return res.source, nil
}

func unixMtime(mtime int64) time.Time {
	if mtime <= 0 {
		return time.Time{}
	}
	return time.Unix(0, mtime) // RefMtime is unix-nanoseconds
}
