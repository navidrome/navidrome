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
	"github.com/navidrome/navidrome/utils"
)

// blurHashState is a decode cache: the hash last computed for an artwork's served bytes, keyed by
// their checksum, so repeated serves of the same image skip the decode.
type blurHashState struct {
	sum  uint64
	hash string
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

// maxDecodePixels bounds the decoded raster: the tee's byte cap limits compressed size only, and a
// small file can declare huge dimensions that would allocate GBs on decode (decompression bomb).
const maxDecodePixels = 36_000_000 // ~6000x6000; decoded RGBA tops out around 144MB

// update hashes the exact bytes served for artID and persists the result. Placeholder bytes mean the
// entity has no artwork anymore, so they clear a stored hash instead. start is when the serve began.
func (u *blurHashUpdater) update(ctx context.Context, artID model.ArtworkID, data []byte, version, start time.Time) {
	// Decoding arbitrary image bytes can panic; the serve already succeeded, so just log it.
	defer func() {
		if r := recover(); r != nil {
			log.Error(ctx, "BlurHash: recovered from panic", "artID", artID, "panic", r)
		}
	}()
	// ArtworkID embeds the client token's LastUpdate; zero it so the decode cache keys by identity.
	artID.LastUpdate = time.Time{}
	// The response is already written when the tee fires; a client abort must not lose the write.
	ctx = context.WithoutCancel(ctx)
	if isPlaceholder(data) {
		u.clearIfStored(ctx, artID)
		return
	}
	sum := checksum(data)
	hash := u.cachedHash(artID, sum)
	if hash == "" {
		cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
		// int64: on 32-bit builds the pixel product can overflow int and bypass the guard.
		if err != nil || int64(cfg.Width)*int64(cfg.Height) > maxDecodePixels {
			// Undecodable or oversized served bytes are not proof of change; keep the stored hash.
			log.Trace(ctx, "BlurHash: skipping served bytes", "artID", artID, "width", cfg.Width, "height", cfg.Height, err)
			return
		}
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			log.Trace(ctx, "BlurHash: served bytes not decodable, keeping stored hash", "artID", artID, err)
			return
		}
		b := img.Bounds()
		x, y := blurhash.Components(b.Dx(), b.Dy())
		if hash, err = blurhash.Encode(img, x, y); err != nil || hash == "" {
			return
		}
	}
	stored, storedAt, entityVersion, err := u.loadState(ctx, artID)
	if err != nil {
		return
	}
	// Clamp the persisted version up to the entity's, but never past the serve's start: a version that
	// predates the serve is provably covered by the served bytes, one that landed mid-serve is not —
	// there the clamp stops, the DTO omits, and the next serve of the new bytes heals.
	if entityVersion.After(start) {
		entityVersion = start
	}
	if stored == hash && storedAt != nil && !storedAt.Before(entityVersion) {
		u.remember(artID, blurHashState{sum: sum, hash: hash})
		return
	}
	target := capAtNow(utils.TimeNewest(version, entityVersion))
	if err := u.persist(ctx, artID, hash, target); err != nil {
		log.Warn(ctx, "BlurHash: error persisting", "artID", artID, err)
		return
	}
	u.remember(artID, blurHashState{sum: sum, hash: hash})
}

// clearIfStored clears the persisted hash after a placeholder was served (a cold map costs one row
// read to skip never-hashed entities); a failed read clears nothing — unknown state is not deletion.
func (u *blurHashUpdater) clearIfStored(ctx context.Context, artID model.ArtworkID) {
	if !eligibleKind(artID) {
		return
	}
	artID.LastUpdate = time.Time{}
	ctx = context.WithoutCancel(ctx)
	u.mutex.Lock()
	prev, ok := u.seen[artID]
	u.mutex.Unlock()
	if ok && prev.hash == "" {
		return
	}
	if !ok {
		stored, _, _, err := u.loadState(ctx, artID)
		if err != nil {
			return
		}
		if stored == "" {
			u.remember(artID, blurHashState{})
			return
		}
	}
	if err := u.persist(ctx, artID, "", time.Now()); err != nil {
		log.Warn(ctx, "BlurHash: error clearing hash", "artID", artID, err)
		return
	}
	u.remember(artID, blurHashState{})
}

// cachedHash returns the previously computed hash when the served bytes are unchanged.
func (u *blurHashUpdater) cachedHash(artID model.ArtworkID, sum uint64) string {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	if prev, ok := u.seen[artID]; ok && prev.hash != "" && prev.sum == sum {
		return prev.hash
	}
	return ""
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

func (u *blurHashUpdater) loadState(ctx context.Context, artID model.ArtworkID) (string, *time.Time, time.Time, error) {
	switch artID.Kind {
	case model.KindAlbumArtwork:
		al, err := u.ds.Album(ctx).Get(artID.ID)
		if err != nil {
			return "", nil, time.Time{}, err
		}
		return al.BlurHash, al.BlurHashUpdatedAt, al.ArtworkUpdatedAt(), nil
	case model.KindArtistArtwork:
		ar, err := u.ds.Artist(ctx).Get(artID.ID)
		if err != nil {
			return "", nil, time.Time{}, err
		}
		return ar.BlurHash, ar.BlurHashUpdatedAt, ar.ArtworkUpdatedAt(), nil
	case model.KindPlaylistArtwork:
		pl, err := u.ds.Playlist(ctx).Get(artID.ID)
		if err != nil {
			return "", nil, time.Time{}, err
		}
		return pl.BlurHash, pl.BlurHashUpdatedAt, pl.ArtworkUpdatedAt(), nil
	}
	return "", nil, time.Time{}, model.ErrNotFound
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
