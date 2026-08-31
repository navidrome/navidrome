package artwork

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/zeebo/xxh3"
)

// ReprocessKinds omits media files: they resolve embedded-only, at scan or on view. Artists lead
// so bulk enqueues give the most external-dependent kind a queue headstart.
var ReprocessKinds = []model.Kind{
	model.KindArtistArtwork, model.KindAlbumArtwork, model.KindPlaylistArtwork, model.KindRadioArtwork,
}

// KeepsState reports whether a kind is recorded in item_artwork and the artwork queue. Disc
// artwork is read through on every request and cached by content key, so it has neither.
func KeepsState(kind model.Kind) bool { return kind != model.KindDiscArtwork }

// RefreshableKinds is every kind Refresh can clear and re-queue, so it holds exactly the kinds
// KeepsState admits. Media files are absent from ReprocessKinds but belong here: the worker
// resolves them, it just never enumerates them in bulk.
var RefreshableKinds = append(slices.Clone(ReprocessKinds), model.KindMediaFileArtwork)

// settlesAbsentOnGiveUp reports whether an exhausted retry budget records an absent state. Media
// files are excluded: a track with no row still falls back to its disc or album art.
func settlesAbsentOnGiveUp(prefix string) bool {
	kind, ok := model.ParseKind(prefix)
	return ok && KeepsState(kind) && kind != model.KindMediaFileArtwork
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

// ReconcileConfigFingerprint warns when the artwork config changed since the library was last
// resolved under it. Nothing re-resolves on its own; applying a change is an explicit reprocess.
func ReconcileConfigFingerprint(ctx context.Context, ds model.DataStore) error {
	current := ConfigFingerprint()
	stored, err := ds.Property(ctx).DefaultGet(consts.ArtConfFingerprintPropertyKey, "")
	if err != nil {
		return err
	}
	switch stored {
	case current:
	case "":
		// An unset fingerprint counts as current; the alternative warns every upgrading install once.
		return MarkConfigApplied(ctx, ds)
	default:
		log.Warn(ctx, "Artwork: Config changed since the last full reprocess. Stored artwork keeps "+
			"the old resolution; run 'navidrome artwork reprocess --all' to apply the change",
			"stored", stored, "current", current, "inputs", FingerprintInputs())
	}
	return nil
}

// MarkConfigApplied records the current fingerprint as the one the library is resolved under.
func MarkConfigApplied(ctx context.Context, ds model.DataStore) error {
	return ds.Property(ctx).Put(consts.ArtConfFingerprintPropertyKey, ConfigFingerprint())
}

// enqueueMissingAll is the safety net for entities a scan never enqueued (added between scans, or scanner off).
func enqueueMissingAll(ctx context.Context, ds model.DataStore) error {
	queue := ds.ArtworkQueue(ctx)
	for _, kind := range ReprocessKinds {
		if _, err := queue.EnqueueAllMissing(kind, model.ArtworkPriorityRecheck); err != nil {
			return err
		}
	}
	return nil
}

// ItemName resolves a kind+id to the entity's display name, and errors when the item
// does not exist. Callers use it to reject ids that would otherwise orphan a queue row.
func ItemName(ctx context.Context, ds model.DataStore, kind model.Kind, id string) (string, error) {
	switch kind {
	case model.KindArtistArtwork:
		ar, err := ds.Artist(ctx).Get(id)
		if err != nil {
			return "", err
		}
		return ar.Name, nil
	case model.KindAlbumArtwork:
		al, err := ds.Album(ctx).Get(id)
		if err != nil {
			return "", err
		}
		return al.Name, nil
	case model.KindPlaylistArtwork:
		pls, err := ds.Playlist(ctx).Get(id)
		if err != nil {
			return "", err
		}
		return pls.Name, nil
	case model.KindRadioArtwork:
		rd, err := ds.Radio(ctx).Get(id)
		if err != nil {
			return "", err
		}
		return rd.Name, nil
	case model.KindMediaFileArtwork:
		mf, err := ds.MediaFile(ctx).Get(id)
		if err != nil {
			return "", err
		}
		return mf.Title, nil
	case model.KindDiscArtwork:
		return discArtworkName(ctx, ds, id)
	}
	return "", fmt.Errorf("unsupported kind %q", kind.Prefix())
}

func discArtworkName(ctx context.Context, ds model.DataStore, id string) (string, error) {
	albumID, discNumber, err := model.ParseDiscArtworkID(id)
	if err != nil {
		return "", err
	}
	al, err := ds.Album(ctx).Get(albumID)
	if err != nil {
		return "", err
	}
	name := fmt.Sprintf("%s (disc %d)", al.Name, discNumber)
	// The subtitle is itself a DiscArtPriority candidate, so name it where the chain can be read against it.
	if subtitle := strings.TrimSpace(al.Discs[discNumber]); subtitle != "" {
		name += ": " + subtitle
	}
	return name, nil
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
