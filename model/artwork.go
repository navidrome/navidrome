package model

import "time"

// Artwork is one unique image, identified by the XXH3-64 hash of its bytes.
type Artwork struct {
	Hash       string    `structs:"hash"`
	Mime       string    `structs:"mime"`
	Width      int       `structs:"width"`
	Height     int       `structs:"height"`
	SizeBytes  int64     `structs:"size_bytes"`
	BlurHash   string    `structs:"blur_hash"`
	SourcePath string    `structs:"source_path"`
	RefMtime   int64     `structs:"ref_mtime"`
	CreatedAt  time.Time `structs:"created_at"`
}

const ImageTypePrimary = "primary"

// ItemArtwork is an entity's resolved artwork state. Hash=="" means known absent.
type ItemArtwork struct {
	ItemKind  string `structs:"item_kind"`
	ItemID    string `structs:"item_id"`
	ImageType string `structs:"image_type"`
	Hash      string `structs:"hash"`
	Source    string `structs:"source"`
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
}

// Absent reports a known-absent artwork state (resolved, no image).
func (i ItemArtworkInfo) Absent() bool { return i.Hash == "" }

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
	GetItemArtwork(kind, id, imageType string) (*ItemArtwork, error)
	PutItemArtwork(ia *ItemArtwork) error
	DeleteForItem(kind, id string) error
	// GetInfoForItems hydrates a page: one batched query, item_artwork joined to artwork.
	GetInfoForItems(kind string, ids []string) (map[string]ItemArtworkInfo, error)
	// GetAllMimes returns hash -> current mime for every stored artwork, for sweep retention checks.
	GetAllMimes() (map[string]string, error)
}

type ArtworkQueueRepository interface {
	// Enqueue upserts; an existing row keeps the higher of the two priorities.
	Enqueue(items ...ArtworkQueueItem) error
	// DequeueBatch returns up to n items with retry_at <= now, priority desc, enqueued_at asc.
	DequeueBatch(n int) ([]ArtworkQueueItem, error)
	// MarkFailed increments attempts and pushes retry_at into the future.
	MarkFailed(kind, id, imageType string, retryAt time.Time) error
	Delete(kind, id, imageType string) error
	Count() (int64, error)
	// EnqueueStaleAbsent inserts queue rows (priority Recheck) for absent states older than cutoff.
	EnqueueStaleAbsent(kind string, attemptedBefore time.Time) (int64, error)
}
