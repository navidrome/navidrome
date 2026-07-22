package artwork

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// FingerprintPropertyKey is the model.PropertyRepository key Backfill compares against
// to detect artwork-affecting config changes across restarts.
const FingerprintPropertyKey = "artwork.fingerprint"

// staleAbsentAge is how old an absent resolution must be before the recheck job retries it.
const staleAbsentAge = 24 * time.Hour

// staleAbsentKinds are the item kinds eligible for the periodic stale-absent recheck.
var staleAbsentKinds = []string{"ar", "al", "pl", "ra"}

// Fingerprint summarizes the config knobs that affect artwork resolution outcomes; a
// change means previously resolved (or absent) state may no longer be correct.
func Fingerprint() string {
	raw := fmt.Sprintf("%s|%s|%s|%s|%t|%s",
		conf.Server.CoverArtPriority, conf.Server.ArtistArtPriority, conf.Server.ArtistImageFolder,
		conf.Server.Agents, conf.Server.EnableExternalServices, consts.Version)
	sum := md5.Sum([]byte(raw)) //nolint:gosec // fingerprint, not security-sensitive
	return hex.EncodeToString(sum[:])
}

// Backfill enqueues artwork resolution for every entity when the config fingerprint changed
// (or was never stored), artists first so those pages resolve before the larger backlog.
func Backfill(ctx context.Context, ds model.DataStore) (bool, error) {
	current := Fingerprint()
	props := ds.Property(ctx)
	stored, err := props.DefaultGet(FingerprintPropertyKey, "")
	if err != nil {
		return false, err
	}
	if stored == current {
		return false, nil
	}

	artists, err := ds.Artist(ctx).GetAll()
	if err != nil {
		return false, err
	}
	if err := enqueueBackfillKind(ctx, ds, "ar", idsOf(artists, func(a model.Artist) string { return a.ID })); err != nil {
		return false, err
	}

	albums, err := ds.Album(ctx).GetAll()
	if err != nil {
		return false, err
	}
	if err := enqueueBackfillKind(ctx, ds, "al", idsOf(albums, func(a model.Album) string { return a.ID })); err != nil {
		return false, err
	}

	playlists, err := ds.Playlist(ctx).GetAll()
	if err != nil {
		return false, err
	}
	if err := enqueueBackfillKind(ctx, ds, "pl", idsOf(playlists, func(p model.Playlist) string { return p.ID })); err != nil {
		return false, err
	}

	radios, err := ds.Radio(ctx).GetAll()
	if err != nil {
		return false, err
	}
	if err := enqueueBackfillKind(ctx, ds, "ra", idsOf(radios, func(r model.Radio) string { return r.ID })); err != nil {
		return false, err
	}

	if err := props.Put(FingerprintPropertyKey, current); err != nil {
		return false, err
	}
	log.Info(ctx, "Artwork: config fingerprint changed, backfill enqueued")
	return true, nil
}

func enqueueBackfillKind(ctx context.Context, ds model.DataStore, kind string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	items := make([]model.ArtworkQueueItem, len(ids))
	for i, id := range ids {
		items[i] = model.ArtworkQueueItem{
			ItemKind: kind, ItemID: id, ImageType: model.ImageTypePrimary, Priority: model.ArtworkPriorityBackfill,
		}
	}
	return ds.ArtworkQueue(ctx).Enqueue(items...)
}

func idsOf[T any](items []T, id func(T) string) []string {
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = id(it)
	}
	return ids
}

// EnqueueStaleAbsentAll requeues absent-state entries older than staleAbsentAge, across
// every artwork-bearing kind, for the periodic recheck job.
func EnqueueStaleAbsentAll(ctx context.Context, ds model.DataStore) error {
	cutoff := time.Now().Add(-staleAbsentAge)
	queue := ds.ArtworkQueue(ctx)
	for _, kind := range staleAbsentKinds {
		if _, err := queue.EnqueueStaleAbsent(kind, cutoff); err != nil {
			return err
		}
	}
	return nil
}
