package persistence

import (
	"context"
	"time"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/pocketbase/dbx"

	. "github.com/Masterminds/squirrel"
)

type podcastRepository struct {
	sqlRepository
	sqlRestful
}

func NewPodcastRepository(ctx context.Context, db dbx.Builder) *podcastRepository {
	r := &podcastRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "podcast"
	return r
}

func (r *podcastRepository) loadEpisodes(podcasts model.Podcasts) error {
	m := map[string]*model.Podcast{}
	var ids []string
	for i := range podcasts {
		podcast := &podcasts[i]
		podcast.PodcastEpisodes = model.PodcastEpisodes{}
		ids = append(ids, podcast.ID)
		m[podcast.ID] = podcast
	}

	return slice.RangeByChunks(ids, 900, func(ids []string) error {
		sql := Select("pe.*").From("podcast_episode pe").
			LeftJoin("podcast on podcast.id = pe.podcast_id").
			OrderBy("podcast_id, pe.publish_date DESC, pe.rowid").
			Where(Eq{"podcast_id": ids})

		var episodes model.PodcastEpisodes
		err := r.queryAll(sql, &episodes)
		if err != nil {
			return err
		}

		for _, episode := range episodes {
			m[episode.PodcastId].PodcastEpisodes = append(m[episode.PodcastId].PodcastEpisodes, episode)
		}

		return nil
	})
}

func (r *podcastRepository) Cleanup() error {
	return r.cleanAnnotations()
}

func (r *podcastRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(options...))
}

func (r *podcastRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	return r.count(Select(), options...)
}

func (r *podcastRepository) DeleteInternal(id string) error {
	return r.delete(Eq{"id": id})
}

func (r *podcastRepository) Delete(id string) error {
	if conf.Server.Podcast.AdminOnly && !r.isAdmin() {
		return rest.ErrPermissionDenied
	}
	return r.delete(Eq{"id": id})
}

func (r *podcastRepository) EntityName() string {
	return "podcast"
}

func (r *podcastRepository) Get(id string, withEpisodes bool) (*model.Podcast, error) {
	sel := r.newSelectWithAnnotation("podcast.id").Columns("podcast.*").Where(Eq{r.tableName + ".id": id})

	res := model.Podcasts{}
	if err := r.queryAll(sel, &res); err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, model.ErrNotFound
	}

	if withEpisodes {
		err := r.loadEpisodes(res)
		return &res[0], err
	}

	return &res[0], nil
}

func (r *podcastRepository) GetAll(withEpisodes bool, options ...model.QueryOptions) (model.Podcasts, error) {
	sel := r.newSelectWithAnnotation("podcast.id", options...).Columns("podcast.*")
	res := model.Podcasts{}
	err := r.queryAll(sel, &res)

	if err != nil {
		return nil, err
	}

	if withEpisodes {
		err = r.loadEpisodes(res)
		if err != nil {
			return nil, err
		}
	}

	return res, err
}

func (r *podcastRepository) NewInstance() interface{} {
	return &model.Podcast{}
}

func (r *podcastRepository) Put(p *model.Podcast) error {
	if conf.Server.Podcast.AdminOnly && !r.isAdmin() {
		return rest.ErrPermissionDenied
	}
	return r.PutInternal(p)
}

func (r *podcastRepository) PutInternal(p *model.Podcast) error {
	if p.ID == "" {
		p.CreatedAt = time.Now()
	}

	p.UpdatedAt = time.Now()
	id, err := r.put(p.ID, p)
	if err != nil {
		return err
	}

	if p.ID == "" {
		p.ID = id
	}
	return nil
}

func (r *podcastRepository) Read(id string) (interface{}, error) {
	return r.Get(id, true)
}

func (r *podcastRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return r.GetAll(false, r.parseRestOptions(options...))
}

var _ model.PodcastRepository = (*podcastRepository)(nil)
