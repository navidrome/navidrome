package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/pocketbase/dbx"
)

type podcastSubscriptionRepository struct {
	sqlRepository
}

func NewPodcastSubscriptionRepository(ctx context.Context, db dbx.Builder) model.PodcastSubscriptionRepository {
	r := &podcastSubscriptionRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "podcast_subscription"
	r.registerModel(&model.PodcastSubscription{}, map[string]filterFunc{
		"channel_id": eqFilter,
		"user_id":    eqFilter,
	})
	return r
}

func (r *podcastSubscriptionRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	return r.count(r.newSelect(), options...)
}

func (r *podcastSubscriptionRepository) Get(id string) (*model.PodcastSubscription, error) {
	sel := r.newSelect().Where(Eq{"id": id}).Columns("*")
	res := model.PodcastSubscription{}
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *podcastSubscriptionRepository) GetAll(options ...model.QueryOptions) (model.PodcastSubscriptions, error) {
	sel := r.newSelect(options...).Columns("*")
	res := model.PodcastSubscriptions{}
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *podcastSubscriptionRepository) FindByChannelAndUser(channelID, userID string) (*model.PodcastSubscription, error) {
	sel := r.newSelect().Where(And{Eq{"channel_id": channelID}, Eq{"user_id": userID}}).Columns("*")
	res := model.PodcastSubscription{}
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *podcastSubscriptionRepository) FindByChannel(channelID string) (model.PodcastSubscriptions, error) {
	sel := r.newSelect().Where(Eq{"channel_id": channelID}).Columns("*")
	res := model.PodcastSubscriptions{}
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *podcastSubscriptionRepository) Put(s *model.PodcastSubscription) error {
	s.UpdatedAt = time.Now()
	if s.ID == "" {
		s.CreatedAt = time.Now()
		s.ID = id.NewRandom()
	}
	_, err := r.put(s.ID, s)
	return err
}

// Delete removes a single subscription (unsubscribe) and reports how many subscriptions remain
// for that subscription's channel afterward, so core/podcasts.Unsubscribe can decide whether the
// now-possibly-orphaned shared channel should be torn down.
func (r *podcastSubscriptionRepository) Delete(subID string) (int64, error) {
	sub, err := r.Get(subID)
	if err != nil {
		return 0, err
	}
	if err := r.delete(Eq{"id": subID}); err != nil {
		return 0, err
	}
	remaining, err := r.count(r.newSelect().Where(Eq{"channel_id": sub.ChannelID}))
	return remaining, err
}

var _ model.PodcastSubscriptionRepository = (*podcastSubscriptionRepository)(nil)
