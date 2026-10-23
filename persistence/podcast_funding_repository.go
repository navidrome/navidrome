package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/pocketbase/dbx"
)

type podcastFundingRepository struct {
	sqlRepository
}

func NewPodcastFundingRepository(db dbx.Builder) model.PodcastFundingRepository {
	r := &podcastFundingRepository{}
	r.db = db
	r.tableName = "podcast_funding"
	r.registerModel(&model.PodcastFundingItem{}, nil)
	return r
}

func (r *podcastFundingRepository) GetByChannel(ctx context.Context, channelID string) (model.PodcastFundingItems, error) {
	sel := r.newSelect(ctx).Columns("*").Where(Eq{"channel_id": channelID})
	var result model.PodcastFundingItems
	err := r.queryAll(ctx, sel, &result)
	return result, err
}

func (r *podcastFundingRepository) GetByChannels(ctx context.Context, channelIDs []string) (model.PodcastFundingItems, error) {
	if len(channelIDs) == 0 {
		return nil, nil
	}
	sel := r.newSelect(ctx).Columns("*").Where(Eq{"channel_id": channelIDs})
	var result model.PodcastFundingItems
	err := r.queryAll(ctx, sel, &result)
	return result, err
}

func (r *podcastFundingRepository) SaveForChannel(ctx context.Context, channelID string, items []model.PodcastFundingItem) error {
	if err := r.delete(ctx, Eq{"channel_id": channelID}); err != nil {
		return err
	}
	now := time.Now()
	for i := range items {
		items[i].ID = id.NewRandom()
		items[i].ChannelID = channelID
		items[i].CreatedAt = now
		if _, err := r.put(ctx, items[i].ID, &items[i]); err != nil {
			return err
		}
	}
	return nil
}

var _ model.PodcastFundingRepository = (*podcastFundingRepository)(nil)
