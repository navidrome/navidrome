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

type podcastEpisodeRepository struct {
	sqlRepository
}

func NewPodcastEpisodeRepository(ctx context.Context, db dbx.Builder) model.PodcastEpisodeRepository {
	r := &podcastEpisodeRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "podcast_episode"
	r.registerModel(&model.PodcastEpisode{}, map[string]filterFunc{
		"title":           containsFilter("podcast_episode.title"),
		"channel_id":      eqFilter,
		"download_status": eqFilter,
	})
	return r
}

// isPermitted gates writes to shared episode infrastructure (download status, file path) to
// admins and headless/system contexts - see the identical comment on
// podcastChannelRepository.isPermitted for why the invalidUserId exemption matters (the
// scheduled refresh/retention job's Put() calls would otherwise be silently rejected).
func (r *podcastEpisodeRepository) isPermitted() bool {
	user := loggedUser(r.ctx)
	return user.IsAdmin || user.ID == invalidUserId
}

// withPlayAnnotation left-joins the current user's annotation row, exposing
// play_count/play_date ("listened") and downloaded/downloaded_at (this user's own "in my
// downloaded list" flag - see model.PodcastEpisode.Downloaded) as one query. Doesn't reuse
// sqlRepository.withAnnotation, which also selects starred/rating/average_rating - podcast
// episodes have no average_rating column and don't support starring/rating (yet).
func (r *podcastEpisodeRepository) withPlayAnnotation(sel SelectBuilder) SelectBuilder {
	userID := loggedUser(r.ctx).ID
	if userID == invalidUserId {
		return sel
	}
	return sel.
		LeftJoin("annotation on ("+
			"annotation.item_id = podcast_episode.id"+
			" AND annotation.item_type = 'podcast_episode'"+
			" AND annotation.user_id = '"+userID+"')").
		Columns("coalesce(play_count, 0) as play_count", "play_date",
			"coalesce(downloaded, false) as downloaded", "downloaded_at")
}

// scopeToSubscriptions restricts an episode query to channels the current user is subscribed to
// - admins and headless/system contexts see every episode regardless of subscription, mirroring
// podcastChannelRepository.selectChannel.
func (r *podcastEpisodeRepository) scopeToSubscriptions(sel SelectBuilder) SelectBuilder {
	user := loggedUser(r.ctx)
	if user.IsAdmin || user.ID == invalidUserId {
		return sel
	}
	return sel.Join("podcast_subscription ps on ps.channel_id = podcast_episode.channel_id and ps.user_id = ?", user.ID)
}

func (r *podcastEpisodeRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	if !r.hasFeaturePermission(model.FeaturePodcasts) {
		return 0, nil
	}
	sql := r.scopeToSubscriptions(r.newSelect())
	return r.count(sql, options...)
}

func (r *podcastEpisodeRepository) Delete(id string) error {
	if !r.isPermitted() {
		return rest.ErrPermissionDenied
	}
	return r.delete(Eq{"id": id})
}

func (r *podcastEpisodeRepository) Get(id string) (*model.PodcastEpisode, error) {
	if !r.hasFeaturePermission(model.FeaturePodcasts) {
		return nil, model.ErrNotFound
	}
	sel := r.scopeToSubscriptions(r.withPlayAnnotation(r.newSelect().Where(Eq{"podcast_episode.id": id}).Columns("podcast_episode.*")))
	res := model.PodcastEpisode{}
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *podcastEpisodeRepository) GetAll(options ...model.QueryOptions) (model.PodcastEpisodes, error) {
	if !r.hasFeaturePermission(model.FeaturePodcasts) {
		return model.PodcastEpisodes{}, nil
	}
	sel := r.scopeToSubscriptions(r.withPlayAnnotation(r.newSelect(options...).Columns("podcast_episode.*")))
	res := model.PodcastEpisodes{}
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *podcastEpisodeRepository) FindByGuid(channelID, guid string) (*model.PodcastEpisode, error) {
	sel := r.newSelect().Where(And{Eq{"channel_id": channelID}, Eq{"guid": guid}}).Columns("*")
	res := model.PodcastEpisode{}
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *podcastEpisodeRepository) GetNewest(count int) (model.PodcastEpisodes, error) {
	if !r.hasFeaturePermission(model.FeaturePodcasts) {
		return model.PodcastEpisodes{}, nil
	}
	sel := r.scopeToSubscriptions(r.withPlayAnnotation(r.newSelect().Columns("podcast_episode.*"))).
		OrderBy("publish_date desc").Limit(uint64(count))
	res := model.PodcastEpisodes{}
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *podcastEpisodeRepository) Put(episode *model.PodcastEpisode, colsToUpdate ...string) error {
	if !r.isPermitted() {
		return rest.ErrPermissionDenied
	}

	episode.UpdatedAt = time.Now()
	if episode.ID == "" {
		episode.CreatedAt = time.Now()
		episode.ID = id.NewRandom()
	}
	if len(colsToUpdate) > 0 {
		colsToUpdate = append(colsToUpdate, "UpdatedAt")
	}
	_, err := r.put(episode.ID, episode, colsToUpdate...)
	return err
}

// ResetPlayCount clears the current user's play_count/play_date annotation for this episode,
// the explicit counterpart to sqlRepository.IncPlayCount (embedded via podcastEpisodeRepository).
func (r *podcastEpisodeRepository) ResetPlayCount(itemID string) error {
	return r.annUpsert(map[string]any{"play_count": 0, "play_date": nil}, itemID)
}

// SetDownloaded sets/clears userID's own "in my downloaded list" flag for the given episodes.
// Takes an explicit userID rather than reusing loggedUser(ctx) the way SetSkip/SetStar do,
// because the caller here is usually the background refresh job fanning downloads out to every
// subscriber according to their own policy, not a single logged-in user acting for themselves -
// see core/podcasts' download-policy fan-out. Does not touch the shared file on disk; that's
// podcast_episode.download_status, managed separately by the actual fetch/delete pipeline.
//
// Uses a real upsert (one statement per id) rather than sqlRepository.annUpsert's update-then-
// insert-if-zero-rows-matched approach, which only handles an all-or-nothing batch correctly -
// with multiple ids, some already having an annotation row and some not (routine for a batch of
// episodes) would leave the ones that already existed silently unupdated.
func (r *podcastEpisodeRepository) SetDownloaded(userID string, downloaded bool, ids ...string) error {
	var downloadedAt any
	if downloaded {
		downloadedAt = time.Now()
	}
	for _, itemID := range ids {
		_, err := r.executeSQL(Expr(
			"INSERT INTO annotation (ann_id, user_id, item_id, item_type, downloaded, downloaded_at) "+
				"VALUES (?, ?, ?, 'podcast_episode', ?, ?) "+
				"ON CONFLICT (user_id, item_id, item_type) DO UPDATE SET downloaded = excluded.downloaded, downloaded_at = excluded.downloaded_at",
			id.NewRandom(), userID, itemID, downloaded, downloadedAt,
		))
		if err != nil {
			return err
		}
	}
	return nil
}

// GetDownloadedForUser returns userID's own downloaded episodes for the given channel, newest
// first - explicitly scoped by the passed-in userID (not loggedUser(ctx)), since the scheduled
// retention job runs against a system context with no single caller to derive that from.
func (r *podcastEpisodeRepository) GetDownloadedForUser(userID, channelID string) (model.PodcastEpisodes, error) {
	sel := r.newSelect().Columns("podcast_episode.*").
		Join("annotation on (annotation.item_id = podcast_episode.id and annotation.item_type = 'podcast_episode' and annotation.user_id = ?)", userID).
		Where(And{
			Eq{"podcast_episode.channel_id": channelID},
			Eq{"annotation.downloaded": true},
		}).
		OrderBy("podcast_episode.publish_date desc")
	res := model.PodcastEpisodes{}
	err := r.queryAll(sel, &res)
	return res, err
}

// AnyUserWantsDownload reports whether any user still has this episode flagged as downloaded -
// deliberately not scoped to the current user's own subscriptions (unlike Get/GetAll), since
// this decides whether the shared file can be deleted for everyone, not just the caller. Queries
// the annotation table directly rather than sqlRepository.exists, which is hardcoded to
// r.tableName ("podcast_episode") - the downloaded flag lives in annotation, not podcast_episode.
func (r *podcastEpisodeRepository) AnyUserWantsDownload(episodeID string) (bool, error) {
	sel := Select("count(*) as exist").From("annotation").Where(And{
		Eq{"item_id": episodeID},
		Eq{"item_type": "podcast_episode"},
		Eq{"downloaded": true},
	})
	var res struct{ Exist int64 }
	err := r.queryOne(sel, &res)
	return res.Exist > 0, err
}

func (r *podcastEpisodeRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(r.ctx, options...))
}

func (r *podcastEpisodeRepository) EntityName() string {
	return "podcastEpisode"
}

func (r *podcastEpisodeRepository) NewInstance() any {
	return &model.PodcastEpisode{}
}

func (r *podcastEpisodeRepository) Read(id string) (any, error) {
	return r.Get(id)
}

func (r *podcastEpisodeRepository) ReadAll(options ...rest.QueryOptions) (any, error) {
	return r.GetAll(r.parseRestOptions(r.ctx, options...))
}

func (r *podcastEpisodeRepository) Save(entity any) (string, error) {
	t := entity.(*model.PodcastEpisode)
	if !r.isPermitted() {
		return "", rest.ErrPermissionDenied
	}
	err := r.Put(t)
	if errors.Is(err, model.ErrNotFound) {
		return "", rest.ErrNotFound
	}
	return t.ID, err
}

func (r *podcastEpisodeRepository) Update(id string, entity any, cols ...string) error {
	t := entity.(*model.PodcastEpisode)
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

var _ model.PodcastEpisodeRepository = (*podcastEpisodeRepository)(nil)
var _ rest.Repository = (*podcastEpisodeRepository)(nil)
var _ rest.Persistable = (*podcastEpisodeRepository)(nil)
