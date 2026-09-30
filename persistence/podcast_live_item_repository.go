package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/pocketbase/dbx"
)

type podcastLiveItemRepository struct {
	sqlRepository
}

func NewPodcastLiveItemRepository(db dbx.Builder) model.PodcastLiveItemRepository {
	r := &podcastLiveItemRepository{}
	r.db = db
	// Must set tableName before registerModel to avoid auto-derived name mismatch.
	r.tableName = "podcast_live_item"
	r.registerModel(&model.PodcastLiveItem{}, nil)
	return r
}

func (r *podcastLiveItemRepository) GetByChannel(ctx context.Context, channelID string) (*model.PodcastLiveItem, error) {
	sel := r.newSelect(ctx).Columns("*").Where(Eq{"channel_id": channelID})
	item := model.PodcastLiveItem{}
	if err := r.queryOne(ctx, sel, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *podcastLiveItemRepository) Upsert(ctx context.Context, item *model.PodcastLiveItem) error {
	existing, err := r.GetByChannel(ctx, item.ChannelID)
	if err == model.ErrNotFound {
		item.ID = id.NewRandom()
		item.CreatedAt = time.Now()
		item.UpdatedAt = time.Now()
		_, err = r.put(ctx, item.ID, item)
		return err
	}
	if err != nil {
		return err
	}
	item.ID = existing.ID
	item.CreatedAt = existing.CreatedAt
	item.UpdatedAt = time.Now()
	_, err = r.put(ctx, item.ID, item)
	return err
}

func (r *podcastLiveItemRepository) DeleteByChannel(ctx context.Context, channelID string) error {
	return r.delete(ctx, Eq{"channel_id": channelID})
}

var _ model.PodcastLiveItemRepository = (*podcastLiveItemRepository)(nil)
