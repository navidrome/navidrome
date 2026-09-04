package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/pocketbase/dbx"
)

type podcastPodrollRepository struct {
	sqlRepository
}

func NewPodcastPodrollRepository(ctx context.Context, db dbx.Builder) model.PodcastPodrollRepository {
	r := &podcastPodrollRepository{}
	r.ctx = ctx
	r.db = db
	// Must set tableName before registerModel to avoid auto-derived name mismatch.
	r.tableName = "podcast_podroll"
	r.registerModel(&model.PodcastPodrollItem{}, nil)
	return r
}

func (r *podcastPodrollRepository) GetByChannel(channelID string) (model.PodcastPodrollItems, error) {
	sel := r.newSelect().Columns("*").Where(Eq{"channel_id": channelID}).OrderBy("sort_order")
	var result model.PodcastPodrollItems
	return result, r.queryAll(sel, &result)
}

func (r *podcastPodrollRepository) GetByChannels(channelIDs []string) (model.PodcastPodrollItems, error) {
	if len(channelIDs) == 0 {
		return nil, nil
	}
	sel := r.newSelect().Columns("*").Where(Eq{"channel_id": channelIDs}).OrderBy("channel_id, sort_order")
	var result model.PodcastPodrollItems
	return result, r.queryAll(sel, &result)
}

func (r *podcastPodrollRepository) SaveForChannel(channelID string, items []model.PodcastPodrollItem) error {
	if err := r.delete(Eq{"channel_id": channelID}); err != nil {
		return err
	}
	now := time.Now()
	for i := range items {
		items[i].ID = id.NewRandom()
		items[i].ChannelID = channelID
		items[i].SortOrder = i
		items[i].CreatedAt = now
		if _, err := r.put(items[i].ID, &items[i]); err != nil {
			return err
		}
	}
	return nil
}

var _ model.PodcastPodrollRepository = (*podcastPodrollRepository)(nil)
