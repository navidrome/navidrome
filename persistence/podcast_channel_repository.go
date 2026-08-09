package persistence

import (
	"context"
	"errors"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/pocketbase/dbx"
)

type podcastChannelRepository struct {
	sqlRepository
}

type dbPodcastChannel struct {
	*model.PodcastChannel `structs:",flatten"`
	SubID                 string     `structs:"-" db:"sub_id"`
	SubDownloadPolicy     string     `structs:"-" db:"sub_download_policy"`
	SubRetentionCount     int        `structs:"-" db:"sub_retention_count"`
	SubRetentionDays      int        `structs:"-" db:"sub_retention_days"`
	SubMaxStorageMB       int        `structs:"-" db:"sub_max_storage_mb"`
	SubCreatedAt          *time.Time `structs:"-" db:"sub_created_at"`
	SubUpdatedAt          *time.Time `structs:"-" db:"sub_updated_at"`
}

func (c *dbPodcastChannel) PostScan() error {
	if c.SubID == "" {
		return nil
	}
	c.PodcastChannel.Subscription = &model.PodcastSubscription{
		ID:             c.SubID,
		ChannelID:      c.PodcastChannel.ID,
		DownloadPolicy: model.PodcastDownloadPolicy(c.SubDownloadPolicy),
		RetentionCount: c.SubRetentionCount,
		RetentionDays:  c.SubRetentionDays,
		MaxStorageMB:   c.SubMaxStorageMB,
	}
	if c.SubCreatedAt != nil {
		c.PodcastChannel.Subscription.CreatedAt = *c.SubCreatedAt
	}
	if c.SubUpdatedAt != nil {
		c.PodcastChannel.Subscription.UpdatedAt = *c.SubUpdatedAt
	}
	return nil
}

type dbPodcastChannels []dbPodcastChannel

func (cs dbPodcastChannels) toModels() model.PodcastChannels {
	res := make(model.PodcastChannels, len(cs))
	for i := range cs {
		res[i] = *cs[i].PodcastChannel
	}
	return res
}

func NewPodcastChannelRepository(ctx context.Context, db dbx.Builder) model.PodcastChannelRepository {
	r := &podcastChannelRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "podcast_channel"
	r.registerModel(&model.PodcastChannel{}, map[string]filterFunc{
		"title":  containsFilter("podcast_channel.title"),
		"status": eqFilter,
	})
	return r
}

// isPermitted gates writes to shared channel infrastructure (feed metadata, refresh state) to
// admins - and to headless/system contexts (invalidUserId), matching the convention every other
// repository in this fork uses for background jobs with no logged-in user (see
// sqlRepository.applyLibraryFilter's admin/invalidUserId bypass). Without that exemption, the
// scheduled podcast refresh job (cmd/root.go's cron callback, which never attaches a user to its
// context) would have every Put() silently rejected with ErrPermissionDenied - new episodes
// discovered by the automatic refresh could never actually be saved. It also lets
// core/podcasts.Unsubscribe delete a now-subscriber-less shared channel on behalf of whichever
// regular user happened to be the last one to unsubscribe, by performing that specific cleanup
// step against a system-context repository instance rather than the requester's own.
func (r *podcastChannelRepository) isPermitted() bool {
	user := loggedUser(r.ctx)
	return user.IsAdmin || user.ID == invalidUserId
}

// selectChannel scopes the channel list to the current user's own subscriptions - admins (and
// headless/system contexts) see every channel regardless of whether they're personally
// subscribed, for management purposes. The caller's own subscription (if any) is always
// left-joined in and exposed via PodcastChannel.Subscription, even for an unrestricted caller.
func (r *podcastChannelRepository) selectChannel(options ...model.QueryOptions) SelectBuilder {
	user := loggedUser(r.ctx)
	userID := user.ID
	sql := r.newSelect(options...).Columns("podcast_channel.*",
		"ps.id as sub_id", "ps.download_policy as sub_download_policy",
		"ps.retention_count as sub_retention_count", "ps.retention_days as sub_retention_days",
		"ps.max_storage_mb as sub_max_storage_mb", "ps.created_at as sub_created_at", "ps.updated_at as sub_updated_at",
	)
	if user.IsAdmin || userID == invalidUserId {
		return sql.LeftJoin("podcast_subscription ps on ps.channel_id = podcast_channel.id and ps.user_id = ?", userID)
	}
	return sql.Join("podcast_subscription ps on ps.channel_id = podcast_channel.id and ps.user_id = ?", userID)
}

func (r *podcastChannelRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	if !r.hasFeaturePermission(model.FeaturePodcasts) {
		return 0, nil
	}
	user := loggedUser(r.ctx)
	sql := r.newSelect()
	if !user.IsAdmin && user.ID != invalidUserId {
		sql = sql.Join("podcast_subscription ps on ps.channel_id = podcast_channel.id and ps.user_id = ?", user.ID)
	}
	return r.count(sql, options...)
}

func (r *podcastChannelRepository) Delete(id string) error {
	if !r.isPermitted() {
		return rest.ErrPermissionDenied
	}
	return r.delete(Eq{"id": id})
}

func (r *podcastChannelRepository) Get(id string) (*model.PodcastChannel, error) {
	if !r.hasFeaturePermission(model.FeaturePodcasts) {
		return nil, model.ErrNotFound
	}
	sel := r.selectChannel().Where(Eq{"podcast_channel.id": id})
	var res dbPodcastChannel
	err := r.queryOne(sel, &res)
	if err != nil {
		return nil, err
	}
	return res.PodcastChannel, nil
}

func (r *podcastChannelRepository) GetAll(options ...model.QueryOptions) (model.PodcastChannels, error) {
	if !r.hasFeaturePermission(model.FeaturePodcasts) {
		return model.PodcastChannels{}, nil
	}
	sel := r.selectChannel(options...)
	var res dbPodcastChannels
	err := r.queryAll(sel, &res)
	if err != nil {
		return nil, err
	}
	return res.toModels(), nil
}

func (r *podcastChannelRepository) GetWithEpisodes(id string) (*model.PodcastChannel, error) {
	channel, err := r.Get(id)
	if err != nil {
		return nil, err
	}
	episodeRepo := NewPodcastEpisodeRepository(r.ctx, r.db)
	episodes, err := episodeRepo.GetAll(model.QueryOptions{
		Filters: Eq{"channel_id": id},
		Sort:    "publish_date",
		Order:   "desc",
	})
	if err != nil {
		return nil, err
	}
	channel.Episodes = episodes
	return channel, nil
}

func (r *podcastChannelRepository) FindByUrl(url string) (*model.PodcastChannel, error) {
	sel := r.selectChannel().Where(Eq{"podcast_channel.url": url})
	var res dbPodcastChannel
	err := r.queryOne(sel, &res)
	if err != nil {
		return nil, err
	}
	return res.PodcastChannel, nil
}

func (r *podcastChannelRepository) Put(channel *model.PodcastChannel, colsToUpdate ...string) error {
	if !r.isPermitted() {
		return rest.ErrPermissionDenied
	}

	channel.UpdatedAt = time.Now()
	if channel.ID == "" {
		channel.CreatedAt = time.Now()
		channel.ID = id.NewRandom()
	}
	if len(colsToUpdate) > 0 {
		colsToUpdate = append(colsToUpdate, "UpdatedAt")
	}
	_, err := r.put(channel.ID, channel, colsToUpdate...)
	return err
}

func (r *podcastChannelRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(r.ctx, options...))
}

func (r *podcastChannelRepository) EntityName() string {
	return "podcastChannel"
}

func (r *podcastChannelRepository) NewInstance() any {
	return &model.PodcastChannel{}
}

func (r *podcastChannelRepository) Read(id string) (any, error) {
	return r.Get(id)
}

func (r *podcastChannelRepository) ReadAll(options ...rest.QueryOptions) (any, error) {
	return r.GetAll(r.parseRestOptions(r.ctx, options...))
}

func (r *podcastChannelRepository) Save(entity any) (string, error) {
	t := entity.(*model.PodcastChannel)
	if !r.isPermitted() {
		return "", rest.ErrPermissionDenied
	}
	err := r.Put(t)
	if errors.Is(err, model.ErrNotFound) {
		return "", rest.ErrNotFound
	}
	return t.ID, err
}

func (r *podcastChannelRepository) Update(id string, entity any, cols ...string) error {
	t := entity.(*model.PodcastChannel)
	t.ID = id
	if !r.isPermitted() {
		return rest.ErrPermissionDenied
	}
	err := r.Put(t)
	if errors.Is(err, model.ErrNotFound) {
		return rest.ErrNotFound
	}
	return err
}

var _ model.PodcastChannelRepository = (*podcastChannelRepository)(nil)
var _ rest.Repository = (*podcastChannelRepository)(nil)
var _ rest.Persistable = (*podcastChannelRepository)(nil)
