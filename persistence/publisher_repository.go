package persistence

import (
	"context"

	"github.com/google/uuid"

	. "github.com/Masterminds/squirrel"
	"github.com/beego/beego/v2/client/orm"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type publisherRepository struct {
	sqlRepository
	sqlRestful
}

func NewPublisherRepository(ctx context.Context, o orm.QueryExecutor) model.PublisherRepository {
	r := &publisherRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "publisher"
	r.filterMappings = map[string]filterFunc{
		"name": containsFilter,
	}
	return r
}

func (r *publisherRepository) GetAll(opt ...model.QueryOptions) (model.Publishers, error) {
	sq := r.newSelect(opt...).Columns("publisher.id", "publisher.name", "a.album_count", "m.song_count").
		LeftJoin("(select ap.publisher_id, count(ap.album_id) as album_count from album_publishers ap group by ap.publisher_id) a on a.publisher_id = publisher.id").
		LeftJoin("(select mg.publisher_id, count(mg.media_file_id) as song_count from media_file_publishers mg group by mg.publisher_id) m on m.publisher_id = publisher.id")
	res := model.Publishers{}
	err := r.queryAll(sq, &res)
	return res, err
}

// Put is an Upsert operation, based on the name of the publisher: If the name already exists, returns its ID, or else
// insert the new publisher in the DB and returns its new created ID.
func (r *publisherRepository) Put(m *model.Publisher) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	sql := Insert("publisher").Columns("id", "name").Values(m.ID, m.Name).
		Suffix("on conflict (name) do update set name=excluded.name returning id")
	resp := model.Publisher{}
	err := r.queryOne(sql, &resp)
	if err != nil {
		return err
	}
	m.ID = resp.ID
	return nil
}

func (r *publisherRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.count(Select(), r.parseRestOptions(options...))
}

func (r *publisherRepository) Read(id string) (interface{}, error) {
	sel := r.newSelect().Columns("*").Where(Eq{"id": id})
	var res model.Publisher
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *publisherRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	sel := r.newSelect(r.parseRestOptions(options...)).Columns("*")
	res := model.Publishers{}
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *publisherRepository) EntityName() string {
	return r.tableName
}

func (r *publisherRepository) NewInstance() interface{} {
	return &model.Publisher{}
}

func (r *publisherRepository) purgeEmpty() error {
	del := Delete(r.tableName).Where(`id in (
select publisher.id from publisher
left join album_publishers ap on publisher.id = ap.publisher_id
left join artist_publishers a on publisher.id = a.publisher_id
left join media_file_publishers mfg on publisher.id = mfg.publisher_id
where ap.publisher_id is null
and a.publisher_id is null
and mfg.publisher_id is null
)`)
	c, err := r.executeSQL(del)
	if err == nil {
		if c > 0 {
			log.Debug(r.ctx, "Purged unused publishers", "totalDeleted", c)
		}
	}
	return err
}

var _ model.PublisherRepository = (*publisherRepository)(nil)
var _ model.ResourceRepository = (*publisherRepository)(nil)
