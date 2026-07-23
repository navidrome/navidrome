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
	Hash        string    // "" for placeholders
	LastUpdated time.Time // zero for placeholders
	Placeholder bool
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
	if errors.Is(err, ErrUnavailable) || errors.Is(err, model.ErrNotFound) {
		return s.placeholder(artID.Kind), nil
	}
	return img, err
}

func (s *service) Get(ctx context.Context, artID model.ArtworkID, size int, square bool) (*Image, error) {
	if artID.ID == "" {
		return nil, ErrUnavailable
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

// serveEntity serves an entity whose state the worker owns (album/artist/playlist/radio):
// found row serves its hash, absent row is unavailable, missing row reads through provisionally.
func (s *service) serveEntity(ctx context.Context, artID model.ArtworkID, size int, square bool) (*Image, error) {
	ia, err := s.ds.Artwork(ctx).GetItemArtwork(artID.Kind.Prefix(), artID.ID, model.ImageTypePrimary)
	switch {
	case errors.Is(err, model.ErrNotFound):
		return s.provisional(ctx, artID, size, square)
	case err != nil:
		return nil, err
	case ia.Hash == "":
		return nil, ErrUnavailable
	default:
		return s.serveHash(ctx, artID, ia, size, square)
	}
}

// serveHash serves the bytes of a found state row: full-size streams the original, sized
// goes through the resize cache. A mismatch/open error is dangling (a warm cache still serves).
func (s *service) serveHash(ctx context.Context, artID model.ArtworkID, ia *model.ItemArtwork, size int, square bool) (*Image, error) {
	art, err := s.ds.Artwork(ctx).GetImage(ia.Hash)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return s.dangling(ctx, artID)
		}
		return nil, err
	}

	if size == 0 && !square {
		rc, err := openOriginal(ia, art.Mime, s.store)
		if err != nil {
			return s.dangling(ctx, artID)
		}
		return &Image{ReadCloser: rc, Hash: ia.Hash, LastUpdated: ia.UpdatedAt}, nil
	}

	item := newResizedItem(ia, art.Mime, size, square, s.store, s.ffmpeg)
	stream, err := s.cache.Get(ctx, item)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		return s.dangling(ctx, artID)
	}
	return &Image{ReadCloser: stream, Hash: ia.Hash, LastUpdated: ia.UpdatedAt}, nil
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
		if ia.RefMtime != 0 && info.ModTime().Unix() != ia.RefMtime {
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
		if info.ModTime().Unix() != ia.RefMtime {
			return nil, errStaleSource
		}
	}
	return store.Open(ia.Hash, mime)
}

// newResizedItem builds the resize-cache reader for a found state row's bytes; shared by
// the serving path and the worker's precache so both key the cache identically.
func newResizedItem(ia *model.ItemArtwork, mime string, size int, square bool, store *ImageStore, ffm ffmpeg.FFmpeg) *resizedItem {
	return &resizedItem{
		hash:       ia.Hash,
		size:       size,
		square:     square,
		lastUpdate: ia.UpdatedAt,
		ffmpeg:     ffm,
		open:       func() (io.ReadCloser, error) { return openOriginal(ia, mime, store) },
	}
}

// provisional does a local-only read-through for an entity with no state row: it enqueues
// the worker (Bump) and serves any local bytes immediately, never writing a state row.
func (s *service) provisional(ctx context.Context, artID model.ArtworkID, size int, square bool) (*Image, error) {
	item := model.ArtworkQueueItem{ItemKind: artID.Kind.Prefix(), ItemID: artID.ID, ImageType: model.ImageTypePrimary}
	res, err := resolveItemLocal(ctx, s.ds, s.ffmpeg, item)
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
	return s.serveBytes(ctx, hash, data, unixMtime(res.refMtime), size, square)
}

// serveBytes serves in-memory bytes: full-size directly, sized through the resize
// cache keyed by the byte-hash (so it lines up with the worker's eventual store entry).
func (s *service) serveBytes(ctx context.Context, hash string, data []byte, lastUpdate time.Time, size int, square bool) (*Image, error) {
	if size == 0 && !square {
		return &Image{ReadCloser: io.NopCloser(bytes.NewReader(data)), Hash: hash, LastUpdated: lastUpdate}, nil
	}
	item := &resizedItem{
		hash:       hash,
		size:       size,
		square:     square,
		lastUpdate: lastUpdate,
		ffmpeg:     s.ffmpeg,
		open:       func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(data)), nil },
	}
	stream, err := s.cache.Get(ctx, item)
	if err != nil {
		return nil, err
	}
	return &Image{ReadCloser: stream, Hash: hash, LastUpdated: lastUpdate}, nil
}

// serveMediaFile serves a track: own found art wins; an absent row delegates to the album;
// a missing row extracts embedded art (if eligible, enqueuing) else delegates without enqueue.
func (s *service) serveMediaFile(ctx context.Context, artID model.ArtworkID, size int, square bool) (*Image, error) {
	ia, err := s.ds.Artwork(ctx).GetItemArtwork("mf", artID.ID, model.ImageTypePrimary)
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
	return s.Get(ctx, mf.AlbumCoverArtID(), size, square)
}

// provisionalEmbedded extracts a track's embedded art for an immediate serve and always
// enqueues the track (Bump) so the worker persists state; it never writes a state row.
func (s *service) provisionalEmbedded(ctx context.Context, artID model.ArtworkID, mf model.MediaFile, size int, square bool) (*Image, error) {
	lib, err := loadLibraryView(ctx, s.ds, mf.LibraryID)
	if err != nil {
		return nil, err
	}
	res, _ := resolveEmbedded(ctx, lib, s.ffmpeg, mf.Path)
	s.enqueue(ctx, artID, model.ArtworkPriorityBump)
	return s.serveResolution(ctx, res, size, square)
}

// serveDisc serves disc-level artwork as a pure provisional read-through: no state rows,
// no enqueue. It tries the disc-folder selection chain and falls back to the album cover.
func (s *service) serveDisc(ctx context.Context, artID model.ArtworkID, size int, square bool) (*Image, error) {
	dr, err := newDiscArtworkReader(ctx, s.ds, artID)
	if err != nil {
		return nil, err
	}
	funcs := dr.fromDiscArtPriority(ctx, s.ffmpeg, conf.Server.DiscArtPriority)
	if r, path, err := selectImageReader(ctx, artID, funcs...); err == nil && r != nil {
		defer r.Close()
		if data, rerr := readCapped(r); rerr == nil {
			if hash, herr := HashImage(bytes.NewReader(data)); herr == nil {
				return s.serveBytes(ctx, hash, data, unixMtime(mtimeViaFS(dr.lib.FS, path)), size, square)
			}
		}
	}
	albumArtID := model.ArtworkID{Kind: model.KindAlbumArtwork, ID: dr.album.ID}
	return s.Get(ctx, albumArtID, size, square)
}

// dangling enqueues a re-resolution at Scan priority and reports the artwork as
// unavailable, leaving the state row untouched.
func (s *service) dangling(ctx context.Context, artID model.ArtworkID) (*Image, error) {
	s.enqueue(ctx, artID, model.ArtworkPriorityScan)
	return nil, ErrUnavailable
}

func (s *service) enqueue(ctx context.Context, artID model.ArtworkID, priority int) {
	err := s.ds.ArtworkQueue(ctx).Enqueue(model.ArtworkQueueItem{
		ItemKind:  artID.Kind.Prefix(),
		ItemID:    artID.ID,
		ImageType: model.ImageTypePrimary,
		Priority:  priority,
	})
	if err != nil {
		log.Warn(ctx, "artwork: could not enqueue re-resolution", "artID", artID, err)
	}
}

func (s *service) placeholder(kind model.Kind) *Image {
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
	return time.Unix(mtime, 0)
}

// resizedItem is an artworkReader that resizes bytes opened by open() and caches the
// result under a hash-derived key.
type resizedItem struct {
	hash       string
	size       int
	square     bool
	lastUpdate time.Time
	ffmpeg     ffmpeg.FFmpeg
	open       func() (io.ReadCloser, error)
}

func (r *resizedItem) Key() string {
	return fmt.Sprintf("h-%s.%d.%v.%s", r.hash, r.size, r.square, formatQualityTag())
}

func (r *resizedItem) LastUpdated() time.Time { return r.lastUpdate }

func (r *resizedItem) Reader(ctx context.Context) (io.ReadCloser, string, error) {
	orig, err := r.open()
	if err != nil {
		return nil, "", err
	}
	defer orig.Close()
	data, err := readCapped(orig)
	if err != nil {
		return nil, "", err
	}
	resized, _, err := resizeImageData(ctx, r.ffmpeg, data, r.size, r.square)
	if err != nil || resized == nil {
		// Resize failed or image already within bounds: serve the original bytes.
		return io.NopCloser(bytes.NewReader(data)), r.Key(), nil
	}
	if rc, ok := resized.(io.ReadCloser); ok {
		return rc, r.Key(), nil
	}
	return io.NopCloser(resized), r.Key(), nil
}
