package persistence

import (
	"context"
	"time"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"

	. "github.com/Masterminds/squirrel"
)

type podcastEpisodeRepository struct {
	sqlRepository
	sqlRestful
}

func NewPodcastEpisodeRepository(ctx context.Context, db dbx.Builder) *podcastEpisodeRepository {
	r := &podcastEpisodeRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "podcast_episode"
	return r
}

func (r *podcastEpisodeRepository) Cleanup() error {
	err := r.cleanAnnotations()
	if err != nil {
		return err
	}

	return r.cleanBookmarks()
}

func (r *podcastEpisodeRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(options...))
}

func (r *podcastEpisodeRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	return r.count(Select(), options...)
}

func (r *podcastEpisodeRepository) Delete(id string) error {
	return r.delete(Eq{"id": id})
}

func (r *podcastEpisodeRepository) EntityName() string {
	return "podcast_episode"
}

func (r *podcastEpisodeRepository) Get(id string) (*model.PodcastEpisode, error) {
	sel := r.newSelectWithAnnotation("podcast_episode.id")
	sel = r.withBookmark(sel, "podcast_episode.id").Where(Eq{"id": id}).Columns("podcast_episode.*")
	res := model.PodcastEpisode{}
	err := r.queryOne(sel, &res)
	return &res, err
}

type onlyGuid struct {
	Guid string `structs:"guid"`
}

func (r *podcastEpisodeRepository) GetAll(options ...model.QueryOptions) (model.PodcastEpisodes, error) {
	sel := r.newSelectWithAnnotation("podcast_episode.id", options...)
	sel = r.withBookmark(sel, "podcast_episode.id").Columns("podcast_episode.*")
	res := model.PodcastEpisodes{}
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *podcastEpisodeRepository) GetEpisodeGuids(id string) (map[string]bool, error) {
	sel := r.newSelect().Columns("guid").Where(Eq{"podcast_id": id})
	res := []onlyGuid{}
	err := r.queryAll(sel, &res)

	if err != nil {
		return nil, err
	}

	mapping := map[string]bool{}
	for _, item := range res {
		mapping[item.Guid] = true
	}

	return mapping, err
}

func (r *podcastEpisodeRepository) GetNewestEpisodes(count int) (model.PodcastEpisodes, error) {
	options := model.QueryOptions{
		Max:   count,
		Order: "desc",
		Sort:  "publish_date",
	}
	return r.GetAll(options)
}

func (r *podcastEpisodeRepository) NewInstance() interface{} {
	return &model.PodcastEpisode{}
}

func (r *podcastEpisodeRepository) Put(p *model.PodcastEpisode) error {
	if p.ID == "" {
		p.CreatedAt = time.Now()
	}

	p.UpdatedAt = time.Now()
	id, err := r.put(p.ID, p)

	if p.ID == "" {
		p.ID = id
	}
	return err
}

func (r *podcastEpisodeRepository) Read(id string) (interface{}, error) {
	return r.Get(id)
}

func (r *podcastEpisodeRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return r.GetAll(r.parseRestOptions(options...))
}

var _ model.PodcastEpisodeRepository = (*podcastEpisodeRepository)(nil)
