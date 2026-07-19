package model

import "time"

// GenreAlias maps a scanned genre name to the canonical name it should be treated as, so
// inconsistently-tagged files (e.g. "Hip-Hop"/"Hip Hop"/"HipHop") collapse into one genre
// everywhere - the genre index, per-genre pages, Subsonic clients, and smart-playlist criteria
// - instead of needing separate merge logic in each of those (which don't all share a common
// "genre ID" concept; some match by tag ID, others by the raw scanned name). Canonicalization is
// applied once, at scan time (see model/metadata's genre alias hook), not as a read-time filter.
type GenreAlias struct {
	ID            string    `structs:"id" json:"id"`
	AliasName     string    `structs:"alias_name" json:"aliasName"`
	CanonicalName string    `structs:"canonical_name" json:"canonicalName"`
	CreatedAt     time.Time `structs:"created_at" json:"createdAt"`
	UpdatedAt     time.Time `structs:"updated_at" json:"updatedAt"`
}

type GenreAliases []GenreAlias

type GenreAliasRepository interface {
	ResourceRepository
	CountAll(options ...QueryOptions) (int64, error)
	Delete(id string) error
	Get(id string) (*GenreAlias, error)
	GetAll(options ...QueryOptions) (GenreAliases, error)
	Put(a *GenreAlias) error
}
