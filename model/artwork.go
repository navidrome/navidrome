package model

import "time"

// Artwork is one unique image, identified by the XXH3-64 hash of its bytes.
type Artwork struct {
	Hash      string    `structs:"hash"`
	Mime      string    `structs:"mime"`
	Width     int       `structs:"width"`
	Height    int       `structs:"height"`
	SizeBytes int64     `structs:"size_bytes"`
	BlurHash  string    `structs:"blur_hash"`
	ThumbHash string    `structs:"thumb_hash"`
	CreatedAt time.Time `structs:"created_at"`
}

const ImageTypePrimary = "primary"

// ItemImage is per-entity artwork state hydrated at query time; never persisted.
type ItemImage struct {
	ImageHash   string `structs:"-" json:"imageHash,omitempty"`
	ImageAbsent bool   `structs:"-" json:"imageAbsent,omitempty"`
	// BlurHash is Jellyfin's; its mappers read this field directly, so it stays off native JSON.
	BlurHash  string `structs:"-" json:"-"`
	ThumbHash string `structs:"-" json:"thumbHash,omitempty"`
	// A thumbhash's own aspect is quantised, so clients need these to shape the placeholder exactly.
	ImageWidth  int `structs:"-" json:"imageWidth,omitempty"`
	ImageHeight int `structs:"-" json:"imageHeight,omitempty"`
}

// AspectRatio is the image's width/height, or nil when the image or its dimensions are unknown.
func (i ItemImage) AspectRatio() *float64 {
	if i.ImageAbsent || i.ImageWidth <= 0 || i.ImageHeight <= 0 {
		return nil
	}
	return new(float64(i.ImageWidth) / float64(i.ImageHeight))
}

// ItemArtwork is an entity's resolved artwork state. Hash=="" means known absent.
type ItemArtwork struct {
	ItemKind  string `structs:"item_kind"`
	ItemID    string `structs:"item_id"`
	ImageType string `structs:"image_type"`
	Hash      string `structs:"hash"`
	Source    string `structs:"source"`
	// SourcePath is the backing file (folder/upload: the image; embedded: the audio file); "" otherwise.
	SourcePath string `structs:"source_path"`
	// RefMtime is SourcePath's mtime (unix-nanoseconds) at resolution; 0 when there is no SourcePath.
	RefMtime int64 `structs:"ref_mtime"`
	// Nullable in the schema, but every insert must set them: these non-pointer fields cannot scan NULL.
	AttemptedAt time.Time `structs:"attempted_at"`
	UpdatedAt   time.Time `structs:"updated_at"`
}

// ItemArtworkInfo is the list-hydration projection (item_artwork joined with artwork).
type ItemArtworkInfo struct {
	ItemID    string
	Hash      string
	BlurHash  string
	ThumbHash string
	Width     int
	Height    int
}

// Absent reports a known-absent artwork state (resolved, no image).
func (i ItemArtworkInfo) Absent() bool { return i.Hash == "" }

// Image projects the hydration entry onto the entity-facing struct.
func (i ItemArtworkInfo) Image() ItemImage {
	return ItemImage{
		ImageHash:   i.Hash,
		ImageAbsent: i.Absent(),
		BlurHash:    i.BlurHash,
		ThumbHash:   i.ThumbHash,
		ImageWidth:  i.Width,
		ImageHeight: i.Height,
	}
}

type ArtworkQueueItem struct {
	ItemKind   string    `structs:"item_kind"`
	ItemID     string    `structs:"item_id"`
	ImageType  string    `structs:"image_type"`
	Priority   int       `structs:"priority"`
	Attempts   int       `structs:"attempts"`
	RetryAt    time.Time `structs:"retry_at"`
	EnqueuedAt time.Time `structs:"enqueued_at"`
}

// Queue priorities: higher drains first.
const (
	ArtworkPriorityRecheck  = 0
	ArtworkPriorityBackfill = 10
	ArtworkPriorityScan     = 50
	ArtworkPriorityBump     = 100
)

// Delete* takes the rows to remove; Purge* finds them itself and reports how many went.
type ArtworkRepository interface {
	GetImage(hash string) (*Artwork, error)
	PutImage(a *Artwork) error
	// PurgeOrphans deletes rows referenced by no item_artwork row and older than cutoff.
	PurgeOrphans(createdBefore time.Time) (int64, error)
	GetItemArtwork(kind Kind, id, imageType string) (*ItemArtwork, error)
	PutItemArtwork(ia *ItemArtwork) error
	DeleteForItems(kind Kind, ids []string) error
	// GetInfoForItems hydrates a page in one batched query.
	GetInfoForItems(kind Kind, ids []string) (map[string]ItemArtworkInfo, error)
	// GetMimeByHash returns hash -> current mime for every stored artwork.
	GetMimeByHash() (map[string]string, error)
	// PurgeDanglingItems removes state rows whose entity no longer exists.
	PurgeDanglingItems() (int64, error)
}

type ArtworkQueueRepository interface {
	// Enqueue upserts; an existing row keeps the higher priority and has its retry_at reset.
	Enqueue(items ...ArtworkQueueItem) error
	// EnqueuePreservingBackoff upserts like Enqueue but preserves an existing row's retry_at, so a
	// request-triggered read-through never resets a failed resolution's backoff.
	EnqueuePreservingBackoff(items ...ArtworkQueueItem) error
	// EnqueueStaleAbsent inserts queue rows (priority Recheck) for absent states older than cutoff.
	EnqueueStaleAbsent(kind Kind, attemptedBefore time.Time) (int64, error)
	// EnqueueAllMissing inserts queue rows for all entities with no item_artwork row, at the given priority.
	EnqueueAllMissing(kind Kind, priority int) (int64, error)
	// EnqueueIfMissing inserts only for items with no item_artwork row yet.
	EnqueueIfMissing(items ...ArtworkQueueItem) error
	// DequeueBatch returns up to n items with retry_at <= now, priority desc, enqueued_at asc.
	// Restricted to the given kinds when any are passed, so one kind cannot block another's drain.
	DequeueBatch(n int, kinds ...string) ([]ArtworkQueueItem, error)
	// MarkFailedIfUnchanged applies the failure backoff only while retry_at still matches
	// seenRetryAt, so a concurrent re-enqueue keeps its fresh eligibility.
	MarkFailedIfUnchanged(kind, id, imageType string, seenRetryAt, retryAt time.Time) error
	// DeleteIfUnchanged deletes only while retry_at still matches, sparing a concurrent re-enqueue.
	DeleteIfUnchanged(kind, id, imageType string, retryAt time.Time) error
	Count() (int64, error)
	// PurgeDangling removes queue rows whose entity no longer exists.
	PurgeDangling() (int64, error)
}
