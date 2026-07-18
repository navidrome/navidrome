package artwork

import (
	"context"
	"errors"
	_ "image/gif"
	"io"
	"time"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/resources"
	"github.com/navidrome/navidrome/utils/cache"
	_ "golang.org/x/image/webp"
)

var ErrUnavailable = errors.New("artwork unavailable")

// maxTeeBytes bounds the per-serve capture buffer; artwork is a few MB, and anything larger is not
// hashed (skipped), so a pathological source can't accumulate unbounded memory across serves.
const maxTeeBytes = 20 * 1024 * 1024

// capAtNow keeps a future artwork mtime (clock skew, a future-stamped file) from being stored as the
// blurhash version, which would let the DTO's !Before check pin the hash until wall time caught up.
func capAtNow(t time.Time) time.Time {
	if now := time.Now(); t.After(now) {
		return now
	}
	return t
}

type Artwork interface {
	Get(ctx context.Context, artID model.ArtworkID, size int, square bool) (io.ReadCloser, time.Time, error)
	GetOrPlaceholder(ctx context.Context, id string, size int, square bool) (io.ReadCloser, time.Time, error)
}

func NewArtwork(ds model.DataStore, cache cache.FileCache, ffmpeg ffmpeg.FFmpeg, provider external.Provider) Artwork {
	a := &artwork{ds: ds, cache: cache, ffmpeg: ffmpeg, provider: provider}
	a.blurHashes = newBlurHashUpdater(a)
	return a
}

// Close stops the background blurhash worker. The server never calls it; tests must, so a leaked
// worker can't touch mocks/filesystems being torn down by the next spec.
func (a *artwork) Close() error {
	if a.blurHashes != nil {
		a.blurHashes.stop()
	}
	return nil
}

type artwork struct {
	ds         model.DataStore
	cache      cache.FileCache
	ffmpeg     ffmpeg.FFmpeg
	provider   external.Provider
	blurHashes *blurHashUpdater
}

type artworkReader interface {
	cache.Item
	LastUpdated() time.Time
	Reader(ctx context.Context) (io.ReadCloser, string, error)
}

func (a *artwork) GetOrPlaceholder(ctx context.Context, id string, size int, square bool) (reader io.ReadCloser, lastUpdate time.Time, err error) {
	artID, err := a.getArtworkId(ctx, id)
	if err == nil {
		reader, lastUpdate, err = a.Get(ctx, artID, size, square)
	}
	if errors.Is(err, ErrUnavailable) {
		if a.blurHashes != nil && eligibleKind(artID) {
			// No bytes flowed through the tee; a real deletion must still clear the stored hash. The
			// worker re-checks so a transient fetch failure doesn't clobber a valid hash.
			a.blurHashes.EnqueueClearIfGone(artID, capAtNow(consts.ServerStart))
		}
		if artID.Kind == model.KindArtistArtwork {
			reader, _ = resources.FS().Open(consts.PlaceholderArtistArt)
		} else {
			reader, _ = resources.FS().Open(consts.PlaceholderAlbumArt)
		}
		return reader, consts.ServerStart, nil
	}
	return reader, lastUpdate, err
}

func (a *artwork) Get(ctx context.Context, artID model.ArtworkID, size int, square bool) (reader io.ReadCloser, lastUpdate time.Time, err error) {
	artReader, err := a.getArtworkReader(ctx, artID, size, square)
	if err != nil {
		return nil, time.Time{}, err
	}

	r, err := a.cache.Get(ctx, artReader)
	if err != nil {
		if !errors.Is(err, context.Canceled) && !errors.Is(err, ErrUnavailable) {
			log.Error(ctx, "Error accessing image cache", "id", artID, "size", size, err)
		}
		return nil, time.Time{}, err
	}
	reader = r
	if a.blurHashes != nil && size == 0 && !square && eligibleKind(artID) {
		// Tee the served bytes: the blurhash is computed from exactly what the client downloads, so it
		// changes precisely when the served cover changes. Placeholder bytes (playlist fallback) clear.
		// The tee wraps r directly, so Close reaches the underlying stream (no fd leak).
		version := capAtNow(artReader.LastUpdated())
		reader = newTeeReader(r, maxTeeBytes,
			func(data []byte) { a.blurHashes.EnqueueBytes(artID, data, version) })
	}
	return reader, artReader.LastUpdated(), nil
}

type coverArtGetter interface {
	CoverArtID() model.ArtworkID
}

func (a *artwork) getArtworkId(ctx context.Context, id string) (model.ArtworkID, error) {
	if id == "" {
		return model.ArtworkID{}, ErrUnavailable
	}
	artID, err := model.ParseArtworkID(id)
	if err == nil {
		return artID, nil
	}

	log.Trace(ctx, "ArtworkID invalid. Trying to figure out kind based on the ID", "id", id)
	entity, err := model.GetEntityByID(ctx, a.ds, id)
	if err != nil {
		return model.ArtworkID{}, err
	}
	if e, ok := entity.(coverArtGetter); ok {
		artID = e.CoverArtID()
	}
	switch e := entity.(type) {
	case *model.Artist:
		log.Trace(ctx, "ID is for an Artist", "id", id, "name", e.Name, "artist", e.Name)
	case *model.Album:
		log.Trace(ctx, "ID is for an Album", "id", id, "name", e.Name, "artist", e.AlbumArtist)
	case *model.MediaFile:
		log.Trace(ctx, "ID is for a MediaFile", "id", id, "title", e.Title, "album", e.Album)
	case *model.Playlist:
		log.Trace(ctx, "ID is for a Playlist", "id", id, "name", e.Name)
	}
	return artID, nil
}

func (a *artwork) getArtworkReader(ctx context.Context, artID model.ArtworkID, size int, square bool) (artworkReader, error) {
	var artReader artworkReader
	var err error
	if size > 0 || square {
		artReader, err = resizedFromOriginal(ctx, a, artID, size, square)
	} else {
		switch artID.Kind {
		case model.KindArtistArtwork:
			artReader, err = newArtistArtworkReader(ctx, a, artID, a.provider)
		case model.KindAlbumArtwork:
			artReader, err = newAlbumArtworkReader(ctx, a, artID, a.provider)
		case model.KindMediaFileArtwork:
			artReader, err = newMediafileArtworkReader(ctx, a, artID)
		case model.KindPlaylistArtwork:
			artReader, err = newPlaylistArtworkReader(ctx, a, artID)
		case model.KindDiscArtwork:
			artReader, err = newDiscArtworkReader(ctx, a, artID)
		case model.KindRadioArtwork:
			artReader, err = newRadioArtworkReader(ctx, a, artID)
		default:
			return nil, ErrUnavailable
		}
	}
	return artReader, err
}
