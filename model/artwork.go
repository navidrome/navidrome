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
	CreatedAt time.Time `structs:"created_at"`
}

const ImageTypePrimary = "primary"

// ItemImage is per-entity artwork state hydrated at query time; never persisted
// (structs:"-" keeps it out of upserts).
type ItemImage struct {
	ImageHash   string `structs:"-" json:"imageHash,omitempty"`
	ImageAbsent bool   `structs:"-" json:"imageAbsent,omitempty"`
	BlurHash    string `structs:"-" json:"blurHash,omitempty"`
	// Dimensions of the original image. A blurhash carries no aspect ratio, so a client
	// needs these to decode the placeholder into the shape the real image will occupy.
	ImageWidth  int `structs:"-" json:"imageWidth,omitempty"`
	ImageHeight int `structs:"-" json:"imageHeight,omitempty"`
}

// AspectRatio is the image's width/height, or nil when there is no image or its dimensions are
// unknown (an unresolved item). Never guesses: a wrong ratio mis-shapes a client's placeholder.
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
	// attempted_at/updated_at are nullable in the schema but always set by PutItemArtwork;
	// raw inserts must set them too, since these non-pointer time.Time fields fail to scan NULL.
	AttemptedAt time.Time `structs:"attempted_at"`
	UpdatedAt   time.Time `structs:"updated_at"`
}

// ItemArtworkInfo is the list-hydration projection (item_artwork joined with artwork).
type ItemArtworkInfo struct {
	ItemID   string
	Hash     string
	BlurHash string
	Width    int
	Height   int
}

// Absent reports a known-absent artwork state (resolved, no image).
func (i ItemArtworkInfo) Absent() bool { return i.Hash == "" }

// Image projects the hydration entry onto the entity-facing struct, so every hydration
// site copies the same set of fields.
func (i ItemArtworkInfo) Image() ItemImage {
	return ItemImage{
		ImageHash:   i.Hash,
		ImageAbsent: i.Absent(),
		BlurHash:    i.BlurHash,
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

type ArtworkRepository interface {
	// Image identity (artwork table)
	GetImage(hash string) (*Artwork, error)
	PutImage(a *Artwork) error
	GetImages(hashes []string) (map[string]Artwork, error)
	// GetOrphanHashes returns hashes referenced by no item_artwork row and older than cutoff.
	GetOrphanHashes(createdBefore time.Time) ([]string, error)
	// DeleteOrphans deletes the given hashes only if still unreferenced and older than cutoff (atomic re-check).
	DeleteOrphans(createdBefore time.Time, hashes []string) error
	// Per-item state (item_artwork table)
	GetItemArtwork(kind Kind, id, imageType string) (*ItemArtwork, error)
	PutItemArtwork(ia *ItemArtwork) error
	DeleteForItem(kind Kind, id string) error
	// DeleteForItems removes state rows for the given ids of one kind, in chunks.
	DeleteForItems(kind Kind, ids []string) error
	// GetInfoForItems hydrates a page: one batched query, item_artwork joined to artwork.
	GetInfoForItems(kind Kind, ids []string) (map[string]ItemArtworkInfo, error)
	// GetAllMimes returns hash -> current mime for every stored artwork, for sweep retention checks.
	GetAllMimes() (map[string]string, error)
	// PurgeDanglingItemArtwork removes state rows whose entity no longer exists.
	PurgeDanglingItemArtwork() (int64, error)
}

type ArtworkQueueRepository interface {
	// Enqueue upserts; an existing row keeps the higher of the two priorities and has its
	// retry_at reset (a detected change wants immediate re-resolution).
	Enqueue(items ...ArtworkQueueItem) error
	// EnqueueBump upserts like Enqueue but preserves an existing row's retry_at, so a
	// request-triggered read-through never resets a failed resolution's backoff.
	EnqueueBump(items ...ArtworkQueueItem) error
	// DequeueBatch returns up to n items with retry_at <= now, priority desc, enqueued_at asc.
	// Restricted to the given item kinds when any are passed, so a drain pool sees only its own
	// work and cannot be held up behind another kind's backlog.
	DequeueBatch(n int, kinds ...string) ([]ArtworkQueueItem, error)
	// MarkFailed increments attempts and pushes retry_at into the future.
	MarkFailed(kind, id, imageType string, retryAt time.Time) error
	// MarkFailedIfUnchanged applies the failure backoff only while retry_at still matches
	// seenRetryAt; a concurrent re-enqueue (which resets retry_at) keeps its fresh eligibility.
	MarkFailedIfUnchanged(kind, id, imageType string, seenRetryAt, retryAt time.Time) error
	Delete(kind, id, imageType string) error
	// DeleteIfUnchanged deletes the row only if its retry_at still matches retryAt, so a
	// concurrent re-enqueue (which resets retry_at) survives instead of being erased.
	DeleteIfUnchanged(kind, id, imageType string, retryAt time.Time) error
	Count() (int64, error)
	// EnqueueStaleAbsent inserts queue rows (priority Recheck) for absent states older than cutoff.
	EnqueueStaleAbsent(kind Kind, attemptedBefore time.Time) (int64, error)
	// EnqueueMissing inserts queue rows (priority Recheck) for entities of the kind that have no
	// item_artwork row at all, so a never-processed entity is eventually resolved even without a scan.
	EnqueueMissing(kind Kind) (int64, error)
	// PurgeDangling removes queue rows whose entity no longer exists.
	PurgeDangling() (int64, error)
}
