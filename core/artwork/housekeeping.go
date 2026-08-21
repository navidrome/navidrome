package artwork

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/zeebo/xxh3"
)

// StaleAbsentAge is how long an absent state is trusted before a recheck retries it.
const StaleAbsentAge = 7 * 24 * time.Hour

// StaleAbsentRecheckBatch caps how many absent states each hourly tick re-queues per kind,
// oldest first, so external agents see a flat drip instead of a daily burst.
const StaleAbsentRecheckBatch = 100

// RecheckKinds omits media files: they resolve embedded-only, at scan or on view.
var RecheckKinds = []model.Kind{
	model.KindArtistArtwork, model.KindAlbumArtwork, model.KindPlaylistArtwork, model.KindRadioArtwork,
}

// KeepsState reports whether a kind is recorded in item_artwork and the artwork queue. Disc
// artwork is read through on every request and cached by content key, so it has neither.
func KeepsState(kind model.Kind) bool { return kind != model.KindDiscArtwork }

// RefreshableKinds is every kind Refresh can clear and re-queue, so it holds exactly the kinds
// KeepsState admits. Media files are absent from RecheckKinds but belong here: the worker
// resolves them, it just never revisits them on its own.
var RefreshableKinds = append(slices.Clone(RecheckKinds), model.KindMediaFileArtwork)

// hasRecheckPath reports whether a periodic job will revisit this kind, making an absent settle recoverable.
func hasRecheckPath(prefix string) bool {
	kind, ok := model.ParseKind(prefix)
	return ok && slices.Contains(RecheckKinds, kind)
}

// artworkEpoch invalidates all resolution state when bumped; bump it whenever resolution semantics change.
const artworkEpoch = 1

// FingerprintInput is one config value the fingerprint covers, named after the setting it came from.
type FingerprintInput struct {
	Name  string
	Value string
}

// FingerprintInputs is the single listing of what ConfigFingerprint hashes.
func FingerprintInputs() []FingerprintInput {
	return []FingerprintInput{
		{"CoverArtPriority", conf.Server.CoverArtPriority},
		{"ArtistArtPriority", conf.Server.ArtistArtPriority},
		{"ArtistImageFolder", conf.Server.ArtistImageFolder},
		{"Agents", conf.Server.Agents},
		{"EnableExternalServices", strconv.FormatBool(conf.Server.EnableExternalServices)},
		{"EnableM3UExternalAlbumArt", strconv.FormatBool(conf.Server.EnableM3UExternalAlbumArt)},
	}
}

// ConfigFingerprint covers the inputs that affect resolution outcomes; a change invalidates stored state.
func ConfigFingerprint() string {
	values := slice.Map(FingerprintInputs(), func(i FingerprintInput) string { return i.Value })
	raw := fmt.Sprintf("%s|%d", strings.Join(values, "|"), artworkEpoch)
	return fmt.Sprintf("%016x", xxh3.Hash([]byte(raw)))
}

// backfillSummary is what a backfill enqueued. MaxExternalLookups is a ceiling, not a forecast:
// an item served by a local source never reaches an agent.
type backfillSummary struct {
	Ran                bool
	PerKind            map[string]int64
	Items              int64
	MaxExternalLookups int64
}

// backfill enqueues artwork resolution for every entity when the config fingerprint changed.
func backfill(ctx context.Context, ds model.DataStore, agentCount func() ImageAgentCount) (backfillSummary, error) {
	start := time.Now()
	ctx = auth.WithAdminUser(ctx, ds)
	current := ConfigFingerprint()
	props := ds.Property(ctx)
	stored, err := props.DefaultGet(consts.ArtConfFingerprintPropertyKey, "")
	if err != nil {
		return backfillSummary{}, err
	}
	if stored == current {
		return backfillSummary{}, nil
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
	// Counted here, not by the caller: building the agent list constructs every enabled agent, and
	// an unchanged fingerprint returns above without ever needing the number.
	agents := agentCount()
	summary := backfillSummary{Ran: true, PerKind: map[string]int64{}}
	for _, k := range kinds {
		ids, err := k.fetch()
		if err != nil {
			return backfillSummary{}, err
		}
		if err := enqueueBackfillKind(ctx, ds, k.kind, ids); err != nil {
			return backfillSummary{}, err
		}
		n := int64(len(ids))
		summary.PerKind[k.kind.Prefix()] = n
		summary.Items += n
		summary.MaxExternalLookups += n * ExternalLookupsPerItem(k.kind, agents)
	}

	if err := props.Put(consts.ArtConfFingerprintPropertyKey, current); err != nil {
		return backfillSummary{}, err
	}
	log.Info(ctx, "Artwork: Config fingerprint changed, backfill enqueued", "items", summary.Items,
		"byKind", summary.PerKind, "maxExternalLookups", summary.MaxExternalLookups,
		"elapsed", time.Since(start))
	return summary, nil
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
	cutoff := time.Now().Add(-StaleAbsentAge)
	queue := ds.ArtworkQueue(ctx)
	for _, kind := range RecheckKinds {
		if _, err := queue.EnqueueStaleAbsent(kind, cutoff, StaleAbsentRecheckBatch); err != nil {
			return err
		}
	}
	return nil
}

// enqueueMissingAll is the safety net for entities a scan never enqueued (added between scans, or scanner off).
func enqueueMissingAll(ctx context.Context, ds model.DataStore) error {
	queue := ds.ArtworkQueue(ctx)
	for _, kind := range RecheckKinds {
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
