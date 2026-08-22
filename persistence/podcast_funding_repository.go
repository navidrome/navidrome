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

func NewPodcastFundingRepository(ctx context.Context, db dbx.Builder) model.PodcastFundingRepository {
	r := &podcastFundingRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "podcast_funding"
	r.registerModel(&model.PodcastFundingItem{}, nil)
	return r
}

func (r *podcastFundingRepository) GetByChannel(channelID string) (model.PodcastFundingItems, error) {
	sel := r.newSelect().Columns("*").Where(Eq{"channel_id": channelID})
	var result model.PodcastFundingItems
	err := r.queryAll(sel, &result)
	return result, err
}

func (r *podcastFundingRepository) GetByChannels(channelIDs []string) (model.PodcastFundingItems, error) {
	if len(channelIDs) == 0 {
		return nil, nil
	}
	sel := r.newSelect().Columns("*").Where(Eq{"channel_id": channelIDs})
	var result model.PodcastFundingItems
	err := r.queryAll(sel, &result)
	return result, err
}

func (r *podcastFundingRepository) SaveForChannel(channelID string, items []model.PodcastFundingItem) error {
	if err := r.delete(Eq{"channel_id": channelID}); err != nil {
		return err
	}
	now := time.Now()
	for i := range items {
		items[i].ID = id.NewRandom()
		items[i].ChannelID = channelID
		items[i].CreatedAt = now
		if _, err := r.put(items[i].ID, &items[i]); err != nil {
			return err
		}
	}
	return nil
}

var _ model.PodcastFundingRepository = (*podcastFundingRepository)(nil)
