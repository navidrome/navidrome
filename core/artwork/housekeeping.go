package artwork

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// FingerprintPropertyKey is the model.PropertyRepository key Backfill compares against
// to detect artwork-affecting config changes across restarts.
const FingerprintPropertyKey = "artwork.fingerprint"

// staleAbsentAge is how old an absent resolution must be before the recheck job retries it.
const staleAbsentAge = 24 * time.Hour

// recheckKinds are the item kinds eligible for the periodic recheck jobs (stale-absent and
// missing-row). Media files are excluded: they resolve embedded-only, at scan or on view.
var recheckKinds = []model.Kind{
	model.KindArtistArtwork, model.KindAlbumArtwork, model.KindPlaylistArtwork, model.KindRadioArtwork,
}

// artworkEpoch invalidates all resolution state when bumped; bump it in the same change that
// alters resolution semantics. Deliberately not the server version, which changes every build.
const artworkEpoch = 1

// Fingerprint summarizes the inputs that affect artwork resolution outcomes; a
// change means previously resolved (or absent) state may no longer be correct.
func Fingerprint() string {
	raw := fmt.Sprintf("%s|%s|%s|%s|%t|%t|%d",
		conf.Server.CoverArtPriority, conf.Server.ArtistArtPriority, conf.Server.ArtistImageFolder,
		conf.Server.Agents, conf.Server.EnableExternalServices, conf.Server.EnableM3UExternalAlbumArt, artworkEpoch)
	sum := md5.Sum([]byte(raw)) //nolint:gosec // fingerprint, not security-sensitive
	return hex.EncodeToString(sum[:])
}

// Backfill enqueues artwork resolution for every entity when the config fingerprint changed
// (or was never stored), artists first so those pages resolve before the larger backlog.
func Backfill(ctx context.Context, ds model.DataStore) (bool, error) {
	ctx = auth.WithAdminUser(ctx, ds)
	current := Fingerprint()
	props := ds.Property(ctx)
	stored, err := props.DefaultGet(FingerprintPropertyKey, "")
	if err != nil {
		return false, err
	}
	if stored == current {
		return false, nil
	}

	// Artists first: few entities, most external-dependent, so they get queue headstart.
	kinds := []struct {
		kind  model.Kind
		fetch func() ([]string, error)
	}{
		{model.KindArtistArtwork, func() ([]string, error) { return ds.Artist(ctx).GetAllIDs() }},
		{model.KindAlbumArtwork, func() ([]string, error) { return ds.Album(ctx).GetAllIDs() }},
		{model.KindPlaylistArtwork, func() ([]string, error) { return ds.Playlist(ctx).GetAllIDs() }},
		{model.KindRadioArtwork, func() ([]string, error) { return ds.Radio(ctx).GetAllIDs() }},
	}
	for _, k := range kinds {
		ids, err := k.fetch()
		if err != nil {
			return false, err
		}
		if err := enqueueBackfillKind(ctx, ds, k.kind, ids); err != nil {
			return false, err
		}
	}

	if err := props.Put(FingerprintPropertyKey, current); err != nil {
		return false, err
	}
	log.Info(ctx, "Artwork: config fingerprint changed, backfill enqueued")
	return true, nil
}

func enqueueBackfillKind(ctx context.Context, ds model.DataStore, kind model.Kind, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	items := make([]model.ArtworkQueueItem, len(ids))
	for i, id := range ids {
		items[i] = model.ArtworkQueueItem{
			ItemKind: kind.Prefix(), ItemID: id, ImageType: model.ImageTypePrimary, Priority: model.ArtworkPriorityBackfill,
		}
	}
	return ds.ArtworkQueue(ctx).Enqueue(items...)
}

// EnqueueStaleAbsentAll requeues absent-state entries older than staleAbsentAge, across
// every artwork-bearing kind, for the periodic recheck job.
func EnqueueStaleAbsentAll(ctx context.Context, ds model.DataStore) error {
	cutoff := time.Now().Add(-staleAbsentAge)
	queue := ds.ArtworkQueue(ctx)
	for _, kind := range recheckKinds {
		if _, err := queue.EnqueueStaleAbsent(kind, cutoff); err != nil {
			return err
		}
	}
	return nil
}

// EnqueueMissingAll requeues entities that have no item_artwork row yet, across every recheck
// kind: the safety net for entities a scan never enqueued (added between scans, or scanner off).
func EnqueueMissingAll(ctx context.Context, ds model.DataStore) error {
	queue := ds.ArtworkQueue(ctx)
	for _, kind := range recheckKinds {
		if _, err := queue.EnqueueMissing(kind); err != nil {
			return err
		}
	}
	return nil
}
