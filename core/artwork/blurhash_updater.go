package artwork

import (
	"context"
	"errors"
	"fmt"
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
// enqueueRequest carries the staleness signals seen at serve time: force (image-cache miss) and
// the reader's LastUpdated, which tracks file mtimes that no entity row timestamp reflects.
type enqueueRequest struct {
	force          bool
	imageUpdatedAt time.Time
}

type blurHashUpdater struct {
	a        *artwork
	mutex    sync.Mutex
	buffer   map[model.ArtworkID]enqueueRequest
	noResult map[model.ArtworkID]time.Time
	wake     chan struct{}
	done     chan struct{}
	runDone  chan struct{}
	started  bool
	stopped  bool
}

func newBlurHashUpdater(a *artwork) *blurHashUpdater {
	return &blurHashUpdater{
		a:        a,
		buffer:   make(map[model.ArtworkID]enqueueRequest),
		noResult: make(map[model.ArtworkID]time.Time),
		wake:     make(chan struct{}, 1),
		done:     make(chan struct{}),
		runDone:  make(chan struct{}),
	}
}

func (u *blurHashUpdater) Enqueue(artID model.ArtworkID, imageUpdatedAt time.Time, force bool) {
	switch artID.Kind {
	case model.KindAlbumArtwork, model.KindArtistArtwork, model.KindPlaylistArtwork:
	default:
		return
	}
	u.mutex.Lock()
	if u.stopped {
		u.mutex.Unlock()
		return
	}
	if !u.started {
		u.started = true
		// Playlist artwork readers require a user in the context. Lazy-starting keeps idle Artwork
		// instances goroutine-free; stop() ends the worker (tests must call it, the server never does).
		go u.run(request.WithUser(context.Background(), model.User{IsAdmin: true}))
	}
	req := u.buffer[artID]
	req.force = req.force || force
	if imageUpdatedAt.After(req.imageUpdatedAt) {
		req.imageUpdatedAt = imageUpdatedAt
	}
	u.buffer[artID] = req
	u.mutex.Unlock()
	select {
	case u.wake <- struct{}{}:
	default:
	}
}

// stop ends the worker and waits for any in-flight computation, so callers can safely tear down
// the resources (DataStore, filesystems) the worker touches.
func (u *blurHashUpdater) stop() {
	u.mutex.Lock()
	if u.stopped {
		u.mutex.Unlock()
		return
	}
	u.stopped = true
	started := u.started
	u.mutex.Unlock()
	close(u.done)
	if started {
		<-u.runDone
	}
}

func (u *blurHashUpdater) run(ctx context.Context) {
	defer close(u.runDone)
	for {
		select {
		case <-u.done:
			return
		case <-u.wake:
		}
		for {
			artID, req, ok := u.next()
			if !ok {
				break
			}
			u.process(ctx, artID, req)
		}
	}
}

func (u *blurHashUpdater) next() (model.ArtworkID, enqueueRequest, bool) {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	for artID, req := range u.buffer {
		delete(u.buffer, artID)
		return artID, req, true
	}
	return model.ArtworkID{}, enqueueRequest{}, false
}

// processTimeout bounds one computation: readers can call external agents, and a hung call must
// not stall the worker forever.
const processTimeout = 30 * time.Second

func (u *blurHashUpdater) process(ctx context.Context, artID model.ArtworkID, req enqueueRequest) {
	// Artwork readers can touch storage, agents and plugins; a panic here must not kill the server.
	defer func() {
		if r := recover(); r != nil {
			log.Error(ctx, "BlurHash: recovered from panic", "artID", artID, "panic", r)
		}
	}()
	ctx, cancel := context.WithTimeout(ctx, processTimeout)
	defer cancel()
	stored, storedAt, version, err := u.loadState(ctx, artID)
	if err != nil {
		log.Trace(ctx, "BlurHash: could not load entity", "artID", artID, err)
		return
	}
	// sig is the newest staleness signal: the entity's artwork version or the served image's own
	// timestamp, whichever is later (file swaps move the latter without touching any row).
	sig := version
	if req.imageUpdatedAt.After(sig) {
		sig = req.imageUpdatedAt
	}
	if !req.force {
		// Current when computed from this row version or later; the snapshot may exceed the row
		// version because file mtimes (which don't move rows) are folded into it on persist.
		if stored != "" && storedAt != nil && !storedAt.Before(version) && !sig.After(*storedAt) {
			return
		}
		if last, ok := u.lastNoResult(artID); ok && !sig.After(last) {
			return
		}
	}
	hash, err := u.computeFromArtwork(ctx, artID)
	if err != nil && !errors.Is(err, ErrUnavailable) {
		// Transient failure (timeout, storage/agent hiccup) — leave un-memoized so a later serve retries.
		log.Trace(ctx, "BlurHash: compute failed", "artID", artID, err)
		return
	}
	if err != nil || hash == "" {
		// Definitively no artwork (or a placeholder) for this state — remember it, so browsing
		// artwork-less entities doesn't re-resolve them on every serve.
		log.Trace(ctx, "BlurHash: nothing to persist", "artID", artID, err)
		u.setNoResult(artID, sig)
		return
	}
	if err := u.persist(ctx, artID, hash, sig); err != nil {
		log.Warn(ctx, "BlurHash: error persisting", "artID", artID, err)
		return
	}
	u.clearNoResult(artID)
}

func (u *blurHashUpdater) lastNoResult(artID model.ArtworkID) (time.Time, bool) {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	t, ok := u.noResult[artID]
	return t, ok
}

// maxNoResultEntries bounds the negative cache; entries only accumulate for artwork-less entities,
// so a wholesale reset just costs those entities one extra verification pass each.
const maxNoResultEntries = 25_000

func (u *blurHashUpdater) setNoResult(artID model.ArtworkID, version time.Time) {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	if len(u.noResult) >= maxNoResultEntries {
		clear(u.noResult)
	}
	u.noResult[artID] = version
}

func (u *blurHashUpdater) clearNoResult(artID model.ArtworkID) {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	delete(u.noResult, artID)
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
	return fmt.Errorf("blurhash: no persister for artwork kind %q", artID.Kind)
}
