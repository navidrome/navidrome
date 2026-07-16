package artwork

import (
	"context"
	"image"
	"sync"
	"time"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/artwork/blurhash"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

// blurHashUpdater keeps stored blurhashes in sync with the artwork actually served. Enqueue is
// cheap (dedup map insert); the worker re-checks freshness against the DB and only decodes when
// the hash is missing or was computed from an older artwork version.
type blurHashUpdater struct {
	a      *artwork
	mutex  sync.Mutex
	buffer map[model.ArtworkID]struct{}
	wake   chan struct{}
}

func newBlurHashUpdater(a *artwork) *blurHashUpdater {
	u := &blurHashUpdater{
		a:      a,
		buffer: make(map[model.ArtworkID]struct{}),
		wake:   make(chan struct{}, 1),
	}
	// Playlist artwork readers require a user in the context.
	ctx := request.WithUser(context.TODO(), model.User{IsAdmin: true})
	go u.run(ctx)
	return u
}

func (u *blurHashUpdater) Enqueue(artID model.ArtworkID) {
	switch artID.Kind {
	case model.KindAlbumArtwork, model.KindArtistArtwork, model.KindPlaylistArtwork:
	default:
		return
	}
	u.mutex.Lock()
	u.buffer[artID] = struct{}{}
	u.mutex.Unlock()
	select {
	case u.wake <- struct{}{}:
	default:
	}
}

func (u *blurHashUpdater) run(ctx context.Context) {
	for range u.wake {
		for {
			artID, ok := u.next()
			if !ok {
				break
			}
			u.process(ctx, artID)
		}
	}
}

func (u *blurHashUpdater) next() (model.ArtworkID, bool) {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	for artID := range u.buffer {
		delete(u.buffer, artID)
		return artID, true
	}
	return model.ArtworkID{}, false
}

func (u *blurHashUpdater) process(ctx context.Context, artID model.ArtworkID) {
	// Artwork readers can touch storage, agents and plugins; a panic here must not kill the server.
	defer func() {
		if r := recover(); r != nil {
			log.Error(ctx, "BlurHash: recovered from panic", "artID", artID, "panic", r)
		}
	}()
	stored, storedAt, version, err := u.loadState(ctx, artID)
	if err != nil {
		log.Trace(ctx, "BlurHash: could not load entity", "artID", artID, err)
		return
	}
	if stored != "" && storedAt != nil && storedAt.Equal(version) {
		return
	}
	hash, err := u.computeFromArtwork(ctx, artID)
	if err != nil || hash == "" {
		log.Trace(ctx, "BlurHash: skipping", "artID", artID, err)
		return
	}
	if err := u.persist(ctx, artID, hash, version); err != nil {
		log.Warn(ctx, "BlurHash: error persisting", "artID", artID, err)
	}
}

func (u *blurHashUpdater) loadState(ctx context.Context, artID model.ArtworkID) (string, *time.Time, time.Time, error) {
	switch artID.Kind {
	case model.KindAlbumArtwork:
		al, err := u.a.ds.Album(ctx).Get(artID.ID)
		if err != nil {
			return "", nil, time.Time{}, err
		}
		return al.BlurHash, al.BlurHashUpdatedAt, al.ArtworkUpdatedAt(), nil
	case model.KindArtistArtwork:
		ar, err := u.a.ds.Artist(ctx).Get(artID.ID)
		if err != nil {
			return "", nil, time.Time{}, err
		}
		return ar.BlurHash, ar.BlurHashUpdatedAt, ar.ArtworkUpdatedAt(), nil
	case model.KindPlaylistArtwork:
		pl, err := u.a.ds.Playlist(ctx).Get(artID.ID)
		if err != nil {
			return "", nil, time.Time{}, err
		}
		return pl.BlurHash, pl.BlurHashUpdatedAt, pl.ArtworkUpdatedAt(), nil
	}
	return "", nil, time.Time{}, model.ErrNotFound
}

func (u *blurHashUpdater) computeFromArtwork(ctx context.Context, artID model.ArtworkID) (string, error) {
	artReader, err := u.a.getArtworkReader(ctx, artID, 0, false)
	if err != nil {
		return "", err
	}
	// Reads straight from the source (not artwork.Get): no re-enqueue, and the returned path
	// identifies placeholder artwork, which must never be persisted as an entity's blurhash.
	r, path, err := artReader.Reader(ctx)
	if err != nil {
		return "", err
	}
	defer r.Close()
	if path == consts.PlaceholderAlbumArt || path == consts.PlaceholderArtistArt {
		return "", nil
	}
	img, _, err := image.Decode(r)
	if err != nil {
		return "", err
	}
	b := img.Bounds()
	x, y := blurhash.Components(b.Dx(), b.Dy())
	return blurhash.Encode(img, x, y)
}

func (u *blurHashUpdater) persist(ctx context.Context, artID model.ArtworkID, hash string, version time.Time) error {
	switch artID.Kind {
	case model.KindAlbumArtwork:
		return u.a.ds.Album(ctx).UpdateBlurHash(artID.ID, hash, version)
	case model.KindArtistArtwork:
		return u.a.ds.Artist(ctx).UpdateBlurHash(artID.ID, hash, version)
	case model.KindPlaylistArtwork:
		return u.a.ds.Playlist(ctx).UpdateBlurHash(artID.ID, hash, version)
	}
	return nil
}
