package artwork

import (
	"bytes"
	"context"
	"fmt"
	"hash/fnv"
	"image"
	"io"
	"sync"
	"time"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/artwork/blurhash"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/resources"
)

// blurHashState remembers what was last persisted for an artwork, keyed by a checksum of the served
// bytes, so repeated serves of the same image skip the decode and the write entirely.
type blurHashState struct {
	sum     uint64
	version time.Time
	hash    string
}

// blurHashUpdater keeps stored blurhashes in sync with the bytes actually served. It runs inline in
// the serving goroutine after the response is fully written (decode+encode is a few ms): the hash is
// a pure function of the captured bytes — no change-detection proxy, no background worker.
type blurHashUpdater struct {
	ds    model.DataStore
	mutex sync.Mutex
	seen  map[model.ArtworkID]blurHashState
}

func newBlurHashUpdater(ds model.DataStore) *blurHashUpdater {
	return &blurHashUpdater{ds: ds, seen: make(map[model.ArtworkID]blurHashState)}
}

func eligibleKind(artID model.ArtworkID) bool {
	switch artID.Kind {
	case model.KindAlbumArtwork, model.KindArtistArtwork, model.KindPlaylistArtwork:
		return true
	}
	return false
}

// update hashes the exact bytes served for artID and persists the result. Placeholder bytes mean the
// entity has no artwork anymore, so they clear a stored hash instead.
func (u *blurHashUpdater) update(ctx context.Context, artID model.ArtworkID, data []byte, version time.Time) {
	// Decoding arbitrary image bytes can panic; the serve already succeeded, so just log it.
	defer func() {
		if r := recover(); r != nil {
			log.Error(ctx, "BlurHash: recovered from panic", "artID", artID, "panic", r)
		}
	}()
	// The response is already written when the tee fires; a client abort must not lose the write.
	ctx = context.WithoutCancel(ctx)
	if isPlaceholder(data) {
		u.clearIfStored(ctx, artID, version)
		return
	}
	sum := checksum(data)
	u.mutex.Lock()
	prev, ok := u.seen[artID]
	u.mutex.Unlock()
	if ok && prev.hash != "" && prev.sum == sum {
		if !version.After(prev.version) {
			return
		}
		// Same bytes under a newer artwork version: re-persist so blur_hash_updated_at keeps pace with
		// the entity version, or the DTO staleness gate would emit the fake after any routine scan.
		u.persistAndRemember(ctx, artID, prev.hash, sum, version)
		return
	}
	img, _, err := image.Decode(bytes.NewReader(data))
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
	u.persistAndRemember(ctx, artID, hash, sum, version)
}

// clearIfStored clears the persisted hash after a placeholder was served (a cold map costs one row
// read to skip never-hashed entities); a failed read clears nothing — unknown state is not deletion.
func (u *blurHashUpdater) clearIfStored(ctx context.Context, artID model.ArtworkID, version time.Time) {
	if !eligibleKind(artID) {
		return
	}
	ctx = context.WithoutCancel(ctx)
	u.mutex.Lock()
	prev, ok := u.seen[artID]
	u.mutex.Unlock()
	if ok && prev.hash == "" {
		return
	}
	if !ok {
		stored, err := u.loadStoredHash(ctx, artID)
		if err != nil {
			return
		}
		if stored == "" {
			u.remember(artID, blurHashState{version: version})
			return
		}
	}
	if err := u.persist(ctx, artID, "", version); err != nil {
		log.Warn(ctx, "BlurHash: error clearing hash", "artID", artID, err)
		return
	}
	u.remember(artID, blurHashState{version: version})
}

func (u *blurHashUpdater) persistAndRemember(ctx context.Context, artID model.ArtworkID, hash string, sum uint64, version time.Time) {
	if err := u.persist(ctx, artID, hash, version); err != nil {
		log.Warn(ctx, "BlurHash: error persisting", "artID", artID, err)
		return
	}
	u.remember(artID, blurHashState{sum: sum, version: version, hash: hash})
}

func (u *blurHashUpdater) remember(artID model.ArtworkID, s blurHashState) {
	u.mutex.Lock()
	u.seen[artID] = s
	u.mutex.Unlock()
}

func checksum(data []byte) uint64 {
	h := fnv.New64a()
	_, _ = h.Write(data)
	return h.Sum64()
}

func (u *blurHashUpdater) loadStoredHash(ctx context.Context, artID model.ArtworkID) (string, error) {
	switch artID.Kind {
	case model.KindAlbumArtwork:
		al, err := u.ds.Album(ctx).Get(artID.ID)
		if err != nil {
			return "", err
		}
		return al.BlurHash, nil
	case model.KindArtistArtwork:
		ar, err := u.ds.Artist(ctx).Get(artID.ID)
		if err != nil {
			return "", err
		}
		return ar.BlurHash, nil
	case model.KindPlaylistArtwork:
		pl, err := u.ds.Playlist(ctx).Get(artID.ID)
		if err != nil {
			return "", err
		}
		return pl.BlurHash, nil
	}
	return "", model.ErrNotFound
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
		return u.ds.Album(ctx).UpdateBlurHash(artID.ID, hash, version)
	case model.KindArtistArtwork:
		return u.ds.Artist(ctx).UpdateBlurHash(artID.ID, hash, version)
	case model.KindPlaylistArtwork:
		return u.ds.Playlist(ctx).UpdateBlurHash(artID.ID, hash, version)
	}
	return fmt.Errorf("blurhash: no persister for artwork kind %q", artID.Kind)
}
