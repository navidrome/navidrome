package artwork

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"slices"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
)

const staleAbsentAge = 24 * time.Hour

// recheckKinds omits media files: they resolve embedded-only, at scan or on view.
var recheckKinds = []model.Kind{
	model.KindArtistArtwork, model.KindAlbumArtwork, model.KindPlaylistArtwork, model.KindRadioArtwork,
}

// hasRecheckPath reports whether a periodic job will revisit this kind, making an absent settle recoverable.
func hasRecheckPath(prefix string) bool {
	kind, ok := model.ParseKind(prefix)
	return ok && slices.Contains(recheckKinds, kind)
}

// artworkEpoch invalidates all resolution state when bumped; bump it whenever resolution semantics change.
const artworkEpoch = 1

// fingerprint covers the inputs that affect resolution outcomes; a change invalidates stored state.
func fingerprint() string {
	raw := fmt.Sprintf("%s|%s|%s|%s|%t|%t|%d",
		conf.Server.CoverArtPriority, conf.Server.ArtistArtPriority, conf.Server.ArtistImageFolder,
		conf.Server.Agents, conf.Server.EnableExternalServices, conf.Server.EnableM3UExternalAlbumArt, artworkEpoch)
	sum := md5.Sum([]byte(raw)) //nolint:gosec // fingerprint, not security-sensitive
	return hex.EncodeToString(sum[:])
}

// backfill enqueues artwork resolution for every entity when the config fingerprint changed.
func backfill(ctx context.Context, ds model.DataStore) (bool, error) {
	start := time.Now()
	ctx = auth.WithAdminUser(ctx, ds)
	current := fingerprint()
	props := ds.Property(ctx)
	stored, err := props.DefaultGet(consts.ArtConfFingerprintPropertyKey, "")
	if err != nil {
		return false, err
	}
	if stored == current {
		return false, nil
	}

	// Artists first: few entities, most external-dependent, so they get a queue headstart.
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

	if err := props.Put(consts.ArtConfFingerprintPropertyKey, current); err != nil {
		return false, err
	}
	log.Info(ctx, "Artwork: Config fingerprint changed, backfill enqueued", "elapsed", time.Since(start))
	return true, nil
}

func enqueueBackfillKind(ctx context.Context, ds model.DataStore, kind model.Kind, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	items := slice.Map(ids, func(id string) model.ArtworkQueueItem {
		return model.ArtworkQueueItem{
			ItemKind: kind.Prefix(), ItemID: id, ImageType: model.ImageTypePrimary, Priority: model.ArtworkPriorityBackfill,
		}
	})
	return ds.ArtworkQueue(ctx).Enqueue(items...)
}

func enqueueStaleAbsentAll(ctx context.Context, ds model.DataStore) error {
	cutoff := time.Now().Add(-staleAbsentAge)
	queue := ds.ArtworkQueue(ctx)
	for _, kind := range recheckKinds {
		if _, err := queue.EnqueueStaleAbsent(kind, cutoff); err != nil {
			return err
		}
	}
	return nil
}

// enqueueMissingAll is the safety net for entities a scan never enqueued (added between scans, or scanner off).
func enqueueMissingAll(ctx context.Context, ds model.DataStore) error {
	queue := ds.ArtworkQueue(ctx)
	for _, kind := range recheckKinds {
		if _, err := queue.EnqueueAllMissing(kind, model.ArtworkPriorityRecheck); err != nil {
			return err
		}
	}
	return nil
}

// Refresh drops an item's resolved artwork state and re-queues it at Bump priority.
func Refresh(ctx context.Context, ds model.DataStore, kind model.Kind, id string) error {
	if err := ds.Artwork(ctx).DeleteForItems(kind, []string{id}); err != nil {
		return fmt.Errorf("clearing artwork state: %w", err)
	}
	item := model.ArtworkQueueItem{ItemKind: kind.Prefix(), ItemID: id, ImageType: model.ImageTypePrimary, Priority: model.ArtworkPriorityBump}
	if err := ds.ArtworkQueue(ctx).Enqueue(item); err != nil {
		return fmt.Errorf("enqueuing artwork refresh: %w", err)
	}
	return nil
}
