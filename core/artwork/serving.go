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
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/resources"
	"github.com/navidrome/navidrome/utils/cache"
)

var ErrUnavailable = errors.New("artwork unavailable")

// errStaleSource signals that a backing file's mtime no longer matches the state
// row's RefMtime: the stored hash may be stale, so the load is aborted (dangling).
var errStaleSource = errors.New("artwork: source file changed since resolution")

// Image is one servable artwork response.
type Image struct {
	io.ReadCloser
	Hash        string    // pixel-identity hash (immutable URL match); "" for placeholders
	ETag        string    // served-representation validator; "" falls back to Hash (full-size original)
	LastUpdated time.Time // zero for placeholders
	Placeholder bool
}

// representationTag identifies a served resized representation for HTTP validation: it changes with
// the dimensions and the encode settings (CoverArtQuality/EnableWebPEncoding), so a config change
// invalidates a revalidating client's cache even though the pixel hash is unchanged.
func representationTag(hash string, size int, square bool) string {
	return fmt.Sprintf("%s.%d.%v.%s", hash, size, square, formatQualityTag())
}

type Service interface {
	// Get serves resolved/provisional artwork; ErrUnavailable or model.ErrNotFound when
	// there is nothing to serve (absent, pending, dangling) — caller picks placeholder vs 404.
	Get(ctx context.Context, artID model.ArtworkID, size int, square bool) (*Image, error)
	// GetOrPlaceholder parses a raw id token (raw entity ids accepted, as today) and falls
	// back to the kind's placeholder image (never resized, Placeholder=true).
	GetOrPlaceholder(ctx context.Context, id string, size int, square bool) (*Image, error)
}

func NewService(ds model.DataStore, cache cache.FileCache, store *ImageStore, ffm ffmpeg.FFmpeg) Service {
	return &service{ds: ds, cache: cache, store: store, ffmpeg: ffm}
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
	// Only a resolvable entity with no art gets the placeholder. An id that matches no entity
	// stays ErrNotFound, so getCoverArt keeps answering error 70 and Jellyfin keeps 404ing.
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
		size = 0 // a negative size is a full-size request, not a giant (OOM) resize rectangle
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

// requestRecheckAge throttles view-triggered rechecks of an absent entity so repeatedly opening a
// genuinely-absent page can't hammer external services; below staleAbsentAge to catch younger absences.
const requestRecheckAge = time.Hour

// serveEntity serves an entity whose state the worker owns (album/artist/playlist/radio):
// found row serves its hash, absent row is unavailable (promoting a stale recheck on view),
// missing row reads through provisionally.
func (s *service) serveEntity(ctx context.Context, artID model.ArtworkID, size int, square bool) (*Image, error) {
	ia, err := s.ds.Artwork(ctx).GetItemArtwork(artID.Kind, artID.ID, model.ImageTypePrimary)
	switch {
	case errors.Is(err, model.ErrNotFound):
		return s.provisional(ctx, artID, size, square)
	case err != nil:
		return nil, err
	case ia.Hash == "":
		// EnqueueBump preserves an existing backoff row's retry_at; for a settled absent row
		// (no queue row) it inserts a fresh, immediately-eligible recheck.
		if time.Since(ia.AttemptedAt) > requestRecheckAge {
			s.enqueue(ctx, artID, model.ArtworkPriorityBump)
		}
		return nil, ErrUnavailable
	default:
		return s.serveHash(ctx, artID, ia, size, square)
	}
}

// serveSource is the one place bytes become an Image: a full-size request streams open()
// directly, anything else goes through the resize cache under key. hash is the pixel identity
// where one exists ("" for disc art, which has no state row) and doubles as the full-size
// validator, so an ETag is only needed when the bytes are resized or the hash is missing.
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

// serveHash serves the bytes of a found state row. A mismatch/open error is dangling (a warm
// cache still serves), but a cancelled request is not: it must not enqueue a re-resolution.
func (s *service) serveHash(ctx context.Context, artID model.ArtworkID, ia *model.ItemArtwork, size int, square bool) (*Image, error) {
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

// openOriginal opens the full-resolution bytes for a found state row, enforcing the
// mtime invariant: bytes are never served under a hash they no longer match.
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
			return nil, errStaleSource
		}
		return f, nil
	}
	// Store-backed (embedded/external/generated): the bytes live in the content-addressed
	// store, but an embedded source still carries the audio file's mtime to detect edits.
	if ia.SourcePath != "" && ia.RefMtime != 0 {
		info, err := os.Stat(ia.SourcePath)
		if err != nil {
			return nil, err
		}
		if info.ModTime().UnixNano() != ia.RefMtime {
			return nil, errStaleSource
		}
	}
	return store.Open(ia.Hash, mime)
}

// provisional does a local-only read-through for an entity with no state row: it enqueues
// the worker (Bump) and serves any local bytes immediately, never writing a state row.
func (s *service) provisional(ctx context.Context, artID model.ArtworkID, size int, square bool) (*Image, error) {
	item := model.ArtworkQueueItem{ItemKind: artID.Kind.Prefix(), ItemID: artID.ID, ImageType: model.ImageTypePrimary}
	res, err := newLocalResolver(s.ds, s.ffmpeg).resolve(ctx, item)
	if err != nil {
		return nil, err
	}
	s.enqueue(ctx, artID, model.ArtworkPriorityBump)
	return s.serveResolution(ctx, res, size, square)
}

// serveResolution turns a local resolution's bytes into a servable Image (byte-hash
// only, no decode). A resolution with no reader is unavailable.
func (s *service) serveResolution(ctx context.Context, res resolution, size int, square bool) (*Image, error) {
	if res.reader == nil {
		return nil, ErrUnavailable
	}
	defer res.reader.Close()
	data, err := readCapped(res.reader)
	if err != nil {
		return nil, ErrUnavailable
	}
	hash, err := HashImage(bytes.NewReader(data))
	if err != nil {
		return nil, ErrUnavailable
	}
	// Keyed by the byte-hash, so the entry lines up with the worker's eventual store entry.
	return s.serveSource(ctx, hash, hash, unixMtime(res.refMtime), size, square,
		func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(data)), nil })
}

// serveMediaFile serves a track: own found art wins; an absent row delegates to the album;
// a missing row extracts embedded art (if eligible, enqueuing) else delegates without enqueue.
func (s *service) serveMediaFile(ctx context.Context, artID model.ArtworkID, size int, square bool) (*Image, error) {
	// Per-track art can be disabled after mf rows were resolved (the setting is not in the
	// config fingerprint). Honor it at serve time so a direct mf- URL falls back to disc/album
	// instead of serving stale persisted embedded art.
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
		// absent row → fall through to album delegation
	case errors.Is(err, model.ErrNotFound):
		// no row → fall through to embedded eligibility / album delegation
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
	// Mirror MediaFile.CoverArtID's fallback: a multi-disc track defers to its disc art
	// (which itself falls back to the album), not straight to the album.
	return s.Get(ctx, mf.DiscCoverArtID(), size, square)
}

// provisionalEmbedded extracts a track's embedded art for an immediate serve and always
// enqueues the track (Bump) so the worker persists state; it never writes a state row.
func (s *service) provisionalEmbedded(ctx context.Context, artID model.ArtworkID, mf model.MediaFile, size int, square bool) (*Image, error) {
	lib, err := loadLibraryView(ctx, s.ds, mf.LibraryID)
	if err != nil {
		return nil, err
	}
	res, ok := resolveEmbedded(ctx, lib, s.ffmpeg, mf.Path)
	s.enqueue(ctx, artID, model.ArtworkPriorityBump)
	if !ok {
		// Eligible but unextractable (truncated frame, no ffmpeg): fall back the way
		// CoverArtID does rather than answer with a placeholder.
		return s.Get(ctx, mf.DiscCoverArtID(), size, square)
	}
	return s.serveResolution(ctx, res, size, square)
}

// serveDisc serves disc-level artwork as a pure provisional read-through: no state rows,
// no enqueue. It tries the disc-folder selection chain and falls back to the album cover.
func (s *service) serveDisc(ctx context.Context, artID model.ArtworkID, size int, square bool) (*Image, error) {
	dr, err := newDiscArtworkReader(ctx, s.ds, artID)
	if err != nil {
		return nil, err
	}
	// Single-disc albums run the chain too: a disc can carry its own art, distinct from the
	// album cover, and DiscArtPriority is what expresses that preference.
	selectImage := func() (io.ReadCloser, string, error) {
		funcs := dr.fromDiscArtPriority(ctx, s.ffmpeg, conf.Server.DiscArtPriority)
		return selectImageReader(ctx, artID, funcs...)
	}
	albumArtID := model.ArtworkID{Kind: model.KindAlbumArtwork, ID: dr.album.ID}
	// Disc art has no state row, so there is no stored content hash — to key the resize cache
	// on, or to fall back to as a validator. Keying on the id and the album's mtime, as the
	// legacy reader did, lets a warm cache answer without touching the filesystem; the chain
	// runs only on a miss. The key carries DiscArtPriority so changing it invalidates.
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

// dangling enqueues a re-resolution at Scan priority and reports the artwork as
// unavailable, leaving the state row untouched.
func (s *service) dangling(ctx context.Context, artID model.ArtworkID) (*Image, error) {
	s.enqueue(ctx, artID, model.ArtworkPriorityScan)
	return nil, ErrUnavailable
}

// enqueue schedules a request-triggered re-resolution. It uses EnqueueBump so an incidental
// read-through never resets a failed resolution's backoff (unlike scan/manual re-resolve).
func (s *service) enqueue(ctx context.Context, artID model.ArtworkID, priority int) {
	err := s.ds.ArtworkQueue(ctx).EnqueueBump(model.ArtworkQueueItem{
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

// PlaceholderFor returns the kind-appropriate placeholder for an artwork id, for callers that must
// serve a placeholder without consulting persisted state (e.g. an access-control denial).
func PlaceholderFor(id string) *Image {
	artID, _ := model.ParseArtworkID(id)
	return placeholderImage(artID.Kind)
}

type coverArtIDGetter interface {
	CoverArtID() model.ArtworkID
}

// parseArtworkID ports the legacy getArtworkId: parse the token, and if it is a raw
// entity id, resolve the entity and take its CoverArtID.
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

func unixMtime(mtime int64) time.Time {
	if mtime <= 0 {
		return time.Time{}
	}
	return time.Unix(0, mtime) // RefMtime is unix-nanoseconds
}

// resizedItem is an artworkReader that resizes bytes opened by open() and caches the
// result under a hash-derived key.
type resizedItem struct {
	hash   string
	size   int
	square bool
	ffmpeg ffmpeg.FFmpeg
	open   func() (io.ReadCloser, error)
}

// Key is the ETag namespaced for the cache, so the validator a client holds and the entry it
// validates can never drift apart.
func (r *resizedItem) Key() string {
	return "h-" + representationTag(r.hash, r.size, r.square)
}

func (r *resizedItem) Reader(ctx context.Context) (io.ReadCloser, error) {
	orig, err := r.open()
	if err != nil {
		return nil, err
	}
	defer orig.Close()
	data, err := readCapped(orig)
	if err != nil {
		return nil, err
	}
	resized, _, err := resizeImageData(ctx, r.ffmpeg, data, r.size, r.square)
	if err != nil || resized == nil {
		// Resize failed or image already within bounds: serve the original bytes.
		return io.NopCloser(bytes.NewReader(data)), nil
	}
	if rc, ok := resized.(io.ReadCloser); ok {
		return rc, nil
	}
	return io.NopCloser(resized), nil
}
