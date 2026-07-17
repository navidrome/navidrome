package artwork

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"io"
	"sync"
	"time"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/artwork/blurhash"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/resources"
)

// enqueueRequest carries the fill-time snapshot (the reader's LastUpdated, which folds row
// timestamps and live file mtimes into one clock). gone marks a serve that failed with
// ErrUnavailable: the only change no passive signal witnesses, so it clears a stale hash.
type enqueueRequest struct {
	snapshot time.Time
	gone     bool
}

// blurHashUpdater keeps stored blurhashes in sync with the artwork entering the image cache.
// Computation is triggered by cache fills (every change), not serves, so the worker only re-derives
// the hash and skips idempotent writes: Enqueue is a cheap dedup insert, a single worker decodes
// and persists.
type blurHashUpdater struct {
	a         *artwork
	mutex     sync.Mutex
	buffer    map[model.ArtworkID]enqueueRequest
	wake      chan struct{}
	done      chan struct{}
	runDone   chan struct{}
	runCancel context.CancelFunc
	started   bool
	stopped   bool
}

func newBlurHashUpdater(a *artwork) *blurHashUpdater {
	return &blurHashUpdater{
		a:       a,
		buffer:  make(map[model.ArtworkID]enqueueRequest),
		wake:    make(chan struct{}, 1),
		done:    make(chan struct{}),
		runDone: make(chan struct{}),
	}
}

// Enqueue schedules a recompute for an original-size cache fill, using the reader's snapshot as the
// artwork version. Called on the miss that fills the cache — i.e. exactly when the served bytes change.
func (u *blurHashUpdater) Enqueue(artID model.ArtworkID, snapshot time.Time) {
	u.enqueue(artID, enqueueRequest{snapshot: snapshot})
}

// EnqueueGone schedules a clear for a serve that failed with ErrUnavailable, so a stored hash stops
// describing artwork that no longer exists (deletion is invisible to every passive signal).
func (u *blurHashUpdater) EnqueueGone(artID model.ArtworkID) {
	u.enqueue(artID, enqueueRequest{gone: true})
}

func (u *blurHashUpdater) enqueue(artID model.ArtworkID, req enqueueRequest) {
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
		// Admin context: playlist artwork readers require a user. Lazy-starting keeps idle Artwork
		// instances goroutine-free; stop() ends the worker (tests must call it, the server never does).
		ctx, cancel := context.WithCancel(request.WithUser(context.Background(), model.User{IsAdmin: true}))
		u.runCancel = cancel
		go u.run(ctx)
	}
	prev := u.buffer[artID]
	if req.snapshot.After(prev.snapshot) {
		prev.snapshot = req.snapshot
	}
	prev.gone = prev.gone || req.gone
	u.buffer[artID] = prev
	u.mutex.Unlock()
	select {
	case u.wake <- struct{}{}:
	default:
	}
}

// stop ends the worker and waits for any in-flight computation, so callers can safely tear down the
// resources (DataStore, filesystems) the worker touches.
func (u *blurHashUpdater) stop() {
	u.mutex.Lock()
	if u.stopped {
		u.mutex.Unlock()
		return
	}
	u.stopped = true
	started := u.started
	cancel := u.runCancel
	u.mutex.Unlock()
	close(u.done)
	if started {
		cancel()
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
			select {
			case <-u.done:
				return
			default:
			}
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

// processTimeout bounds one computation: readers can call external agents, and a hung call must not
// stall the worker forever.
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
	if req.gone {
		// Only the failed serve witnesses a deletion; clear a stored hash so the DTO falls back to
		// the rotating fake. An empty hash means there is nothing to clear.
		if stored != "" {
			if err := u.persist(ctx, artID, "", version); err != nil {
				log.Warn(ctx, "BlurHash: error clearing stale hash", "artID", artID, err)
			}
		}
		return
	}
	// snapshot folds row timestamps and file mtimes into one clock; a stored hash at or after it is
	// already current. This is the only freshness comparison the fill trigger needs.
	if stored != "" && storedAt != nil && !storedAt.Before(req.snapshot) {
		return
	}
	hash, err := u.computeFromArtwork(ctx, artID)
	if err != nil || hash == "" {
		log.Trace(ctx, "BlurHash: nothing to persist", "artID", artID, err)
		// Reaching compute with a stored hash means the cover became a placeholder or vanished;
		// clear it so the DTO stops describing artwork no longer served.
		if stored != "" {
			if err := u.persist(ctx, artID, "", req.snapshot); err != nil {
				log.Warn(ctx, "BlurHash: error clearing stale hash", "artID", artID, err)
			}
		}
		return
	}
	// Unchanged hash with an unmoved snapshot needs no write — keeps cache-disabled installs (which
	// fill on every original serve) from hammering the DB.
	if hash == stored && storedAt != nil && !req.snapshot.After(*storedAt) {
		return
	}
	if err := u.persist(ctx, artID, hash, req.snapshot); err != nil {
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
	// Reads via the cache (not artwork.Get, so no re-enqueue): generated playlist mosaics are
	// random per generation, and the hash must describe the bytes clients actually download.
	r, err := u.a.cache.Get(ctx, artReader)
	if err != nil {
		return "", err
	}
	defer r.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	if isPlaceholder(data) {
		return "", nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	b := img.Bounds()
	x, y := blurhash.Components(b.Dx(), b.Dy())
	return blurhash.Encode(img, x, y)
}

// isPlaceholder byte-compares against the embedded placeholder assets: placeholder artwork must
// never be persisted as an entity's blurhash, and cached reads carry no source path to check.
func isPlaceholder(data []byte) bool {
	for _, p := range placeholderImages() {
		if bytes.Equal(data, p) {
			return true
		}
	}
	return false
}

var placeholderImages = sync.OnceValue(func() [][]byte {
	var imgs [][]byte
	for _, name := range []string{consts.PlaceholderAlbumArt, consts.PlaceholderArtistArt} {
		if f, err := resources.FS().Open(name); err == nil {
			if data, err := io.ReadAll(f); err == nil {
				imgs = append(imgs, data)
			}
			_ = f.Close()
		}
	}
	return imgs
})

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
