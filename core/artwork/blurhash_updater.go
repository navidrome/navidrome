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

// blurHashJob is a unit of work: either bytes to hash (data != nil) or a deletion check (checkGone).
type blurHashJob struct {
	data      []byte
	version   time.Time
	checkGone bool
}

// blurHashUpdater keeps stored blurhashes in sync with the bytes actually served. The serve path tees
// the served image and hands it here; there is no change-detection proxy — the hash is a pure function
// of the captured bytes. A single worker decodes, encodes, and writes (dedup'd in memory).
type blurHashUpdater struct {
	a         *artwork
	mutex     sync.Mutex
	buffer    map[model.ArtworkID]blurHashJob
	last      map[model.ArtworkID]string // last hash written this process; avoids redundant writes
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
		buffer:  make(map[model.ArtworkID]blurHashJob),
		last:    make(map[model.ArtworkID]string),
		wake:    make(chan struct{}, 1),
		done:    make(chan struct{}),
		runDone: make(chan struct{}),
	}
}

func eligibleKind(artID model.ArtworkID) bool {
	switch artID.Kind {
	case model.KindAlbumArtwork, model.KindArtistArtwork, model.KindPlaylistArtwork:
		return true
	}
	return false
}

// EnqueueBytes schedules a blurhash update computed from the exact bytes served for artID.
func (u *blurHashUpdater) EnqueueBytes(artID model.ArtworkID, data []byte, version time.Time) {
	u.enqueue(artID, blurHashJob{data: data, version: version})
}

// EnqueueClearIfGone schedules a deletion check: the worker re-reads the source once and clears the
// stored hash only if it still fails/serves a placeholder, so a transient failure won't clobber it.
func (u *blurHashUpdater) EnqueueClearIfGone(artID model.ArtworkID, version time.Time) {
	u.enqueue(artID, blurHashJob{checkGone: true, version: version})
}

func (u *blurHashUpdater) enqueue(artID model.ArtworkID, job blurHashJob) {
	if !eligibleKind(artID) {
		return
	}
	u.mutex.Lock()
	if u.stopped {
		u.mutex.Unlock()
		return
	}
	if !u.started {
		u.started = true
		// Admin context: playlist artwork readers require a user. Lazy start keeps idle Artwork
		// instances goroutine-free; stop() ends the worker (tests call it, the server never does).
		ctx, cancel := context.WithCancel(request.WithUser(context.Background(), model.User{IsAdmin: true}))
		u.runCancel = cancel
		go u.run(ctx)
	}
	// A bytes job supersedes a pending gone-check (a successful serve proves the artwork exists);
	// otherwise keep whichever is newer.
	prev, ok := u.buffer[artID]
	if !ok || job.data != nil || (prev.checkGone && job.version.After(prev.version)) {
		u.buffer[artID] = job
	}
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
			artID, job, ok := u.next()
			if !ok {
				break
			}
			u.processJob(ctx, artID, job)
		}
	}
}

func (u *blurHashUpdater) next() (model.ArtworkID, blurHashJob, bool) {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	for artID, job := range u.buffer {
		delete(u.buffer, artID)
		return artID, job, true
	}
	return model.ArtworkID{}, blurHashJob{}, false
}

// processTimeout bounds one computation: readers can call external agents, and a hung call must not
// stall the worker forever.
const processTimeout = 30 * time.Second

func (u *blurHashUpdater) processJob(ctx context.Context, artID model.ArtworkID, job blurHashJob) {
	// Artwork readers can touch storage, agents and plugins; a panic here must not kill the server.
	defer func() {
		if r := recover(); r != nil {
			log.Error(ctx, "BlurHash: recovered from panic", "artID", artID, "panic", r)
		}
	}()
	ctx, cancel := context.WithTimeout(ctx, processTimeout)
	defer cancel()

	if job.checkGone {
		u.processGone(ctx, artID, job.version)
		return
	}
	if isPlaceholder(job.data) {
		u.clear(ctx, artID, job.version)
		return
	}
	img, _, err := image.Decode(bytes.NewReader(job.data))
	if err != nil {
		// Undecodable served bytes are not proof of change; leave the stored hash intact.
		log.Trace(ctx, "BlurHash: served bytes not decodable, keeping stored hash", "artID", artID, err)
		return
	}
	b := img.Bounds()
	x, y := blurhash.Components(b.Dx(), b.Dy())
	hash, err := blurhash.Encode(img, x, y)
	if err != nil || hash == "" {
		return
	}
	u.write(ctx, artID, hash, job.version)
}

// processGone re-reads the source once; if it still yields a placeholder or fails, the artwork is
// really gone and the stored hash is cleared. A transient failure recovers by now and is left alone.
func (u *blurHashUpdater) processGone(ctx context.Context, artID model.ArtworkID, version time.Time) {
	artReader, err := u.a.getArtworkReader(ctx, artID, 0, false)
	if err == nil {
		r, gErr := u.a.cache.Get(ctx, artReader)
		if gErr == nil {
			data, rErr := io.ReadAll(r)
			_ = r.Close()
			if rErr == nil && !isPlaceholder(data) {
				return // source came back (or never really failed): keep the hash
			}
		}
	}
	u.clear(ctx, artID, version)
}

func (u *blurHashUpdater) write(ctx context.Context, artID model.ArtworkID, hash string, version time.Time) {
	u.mutex.Lock()
	if u.last[artID] == hash {
		u.mutex.Unlock()
		return
	}
	u.mutex.Unlock()
	if err := u.persist(ctx, artID, hash, version); err != nil {
		log.Warn(ctx, "BlurHash: error persisting", "artID", artID, err)
		return
	}
	u.mutex.Lock()
	u.last[artID] = hash
	u.mutex.Unlock()
}

func (u *blurHashUpdater) clear(ctx context.Context, artID model.ArtworkID, version time.Time) {
	// No cold-map dedup: an empty u.last[artID] means "never written" as easily as "already cleared",
	// so skipping would leave a previous process's DB hash describing gone artwork. Clears are rare.
	if err := u.persist(ctx, artID, "", version); err != nil {
		log.Warn(ctx, "BlurHash: error clearing hash", "artID", artID, err)
		return
	}
	u.mutex.Lock()
	u.last[artID] = ""
	u.mutex.Unlock()
}

// isPlaceholder byte-compares against the embedded placeholder assets: placeholder artwork must never
// be persisted as an entity's blurhash, and captured bytes carry no source path to check.
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
