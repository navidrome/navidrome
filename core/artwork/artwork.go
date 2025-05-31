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

type Artwork interface {
	Get(ctx context.Context, artID model.ArtworkID, size int, square bool) (io.ReadCloser, time.Time, error)
	GetOrPlaceholder(ctx context.Context, id string, size int, square bool) (io.ReadCloser, time.Time, error)
}

func NewArtwork(ds model.DataStore, cache cache.FileCache, ffmpeg ffmpeg.FFmpeg, provider external.Provider) Artwork {
	return &artwork{ds: ds, cache: cache, ffmpeg: ffmpeg, provider: provider}
}

type artwork struct {
	ds       model.DataStore
	cache    cache.FileCache
	ffmpeg   ffmpeg.FFmpeg
	provider external.Provider
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
	return r, artReader.LastUpdated(), nil
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
		default:
			return nil, ErrUnavailable
		}
	}
	return artReader, err
}
