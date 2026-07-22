package persistence

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type artworkRepository struct {
	sqlRepository
}

func NewArtworkRepository(ctx context.Context, db dbx.Builder) model.ArtworkRepository {
	r := &artworkRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "artwork"
	return r
}

func (r *artworkRepository) Get(hash string) (*model.Artwork, error) {
	return nil, model.ErrNotFound
}

func (r *artworkRepository) Put(a *model.Artwork) error {
	return nil
}

func (r *artworkRepository) GetBatch(hashes []string) (map[string]model.Artwork, error) {
	return nil, nil
}

func (r *artworkRepository) GetOrphanHashes(createdBefore time.Time) ([]string, error) {
	return nil, nil
}

func (r *artworkRepository) Delete(hashes ...string) error {
	return nil
}

var _ model.ArtworkRepository = (*artworkRepository)(nil)
