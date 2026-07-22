package persistence

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type artworkQueueRepository struct {
	sqlRepository
}

func NewArtworkQueueRepository(ctx context.Context, db dbx.Builder) model.ArtworkQueueRepository {
	r := &artworkQueueRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "artwork_queue"
	return r
}

func (r *artworkQueueRepository) Enqueue(items ...model.ArtworkQueueItem) error {
	return nil
}

func (r *artworkQueueRepository) DequeueBatch(n int) ([]model.ArtworkQueueItem, error) {
	return nil, nil
}

func (r *artworkQueueRepository) MarkFailed(kind, id, imageType string, retryAt time.Time) error {
	return nil
}

func (r *artworkQueueRepository) Delete(kind, id, imageType string) error {
	return nil
}

func (r *artworkQueueRepository) Count() (int64, error) {
	return 0, nil
}

var _ model.ArtworkQueueRepository = (*artworkQueueRepository)(nil)
