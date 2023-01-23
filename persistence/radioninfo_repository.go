package persistence

import (
	"context"
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/beego/beego/v2/client/orm"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
)

const (
	radioTableName = "radioinfo"
)

type radioInfoRepository struct {
	sqlRepository
	sqlRestful
}

func NewRadioInfoRepository(ctx context.Context, o orm.QueryExecutor) model.RadioInfoRepository {
	r := &radioInfoRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = radioTableName
	r.filterMappings = map[string]filterFunc{
		"country":  countryFilter,
		"existing": existsFilter,
		"id":       idFilter(r.tableName),
		"https":    httpsFilter,
		"name":     nameFilter,
		"tags":     tagsFilter,
	}
	return r
}

func toBool(value interface{}) bool {
	boolResult, ok := value.(bool)

	if !ok {
		text := value.(string)
		boolResult = text == "true"
	}

	return boolResult
}

func countryFilter(field string, value interface{}) Sqlizer {
	country := value.(string)
	return Like{radioTableName + ".country": "%" + country + "%"}
}

func existsFilter(field string, value interface{}) Sqlizer {
	existing := toBool(value)
	return Eq{"existing": existing}
}

func httpsFilter(field string, value interface{}) Sqlizer {
	ishttps := toBool(value)

	if ishttps {
		return Like{radioTableName + ".url": "https://%"}
	} else {
		return Like{radioTableName + ".url": "http://%"}
	}
}

func nameFilter(field string, value interface{}) Sqlizer {
	name := value.(string)
	return Like{radioTableName + ".name": "%" + name + "%"}
}

func tagsFilter(field string, value interface{}) Sqlizer {
	parts := strings.Split(value.(string), ",")
	filters := And{}
	for _, part := range parts {
		filters = append(filters, Like{radioTableName + ".tags": "%" + strings.TrimSpace(part) + "%"})
	}
	return filters
}

func (r *radioInfoRepository) baseQuery(options ...model.QueryOptions) SelectBuilder {
	return r.newSelect(options...).
		Column("radioinfo.*, (r.id is not null) AS existing").
		LeftJoin("radio r ON r.radioinfo_id = radioinfo.id")
}

func (r *radioInfoRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	return r.count(r.baseQuery(), options...)
}

func (r *radioInfoRepository) Insert(m *model.RadioInfo) error {
	radioMap, _ := toSqlArgs(*m, m.BaseRadioInfo)

	delete(radioMap, "existing")

	sql := Insert(r.tableName).SetMap(radioMap)
	_, err := r.executeSQL(sql)
	return err
}

func (r *radioInfoRepository) Get(id string) (*model.RadioInfo, error) {
	sel := r.baseQuery().
		Where(Eq{"radioinfo.id": id})
	res := model.RadioInfo{}
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *radioInfoRepository) GetAll(options ...model.QueryOptions) (model.RadioInfos, error) {
	sel := r.baseQuery(options...)
	res := model.RadioInfos{}
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *radioInfoRepository) GetAllIds() (map[string]bool, error) {
	sel := r.newSelect().Columns("id").OrderBy("id")
	res := []string{}
	err := r.queryAll(sel, &res)

	if err != nil {
		return nil, err
	}

	mapping := map[string]bool{}

	for _, key := range res {
		mapping[key] = true
	}
	return mapping, nil
}

func (r *radioInfoRepository) Update(m *model.RadioInfo) error {
	radioMap, _ := toSqlArgs(*m, m.BaseRadioInfo)

	delete(radioMap, "existing")

	sql := Update(r.tableName).SetMap(radioMap).Where(Eq{"id": m.ID})
	_, err := r.executeSQL(sql)
	return err
}

func (r *radioInfoRepository) DeleteMany(id []string) error {
	query := Delete(r.tableName).Where(Eq{"id": id})
	_, err := r.executeSQL(query)
	return err
}

func (r *radioInfoRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(options...))
}

func (r *radioInfoRepository) EntityName() string {
	return "radioinfo"
}

func (r *radioInfoRepository) NewInstance() interface{} {
	return &model.RadioInfo{}
}

func (r *radioInfoRepository) Read(id string) (interface{}, error) {
	return r.Get(id)
}

func (r *radioInfoRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return r.GetAll(r.parseRestOptions(options...))
}

var _ model.RadioInfoRepository = (*radioInfoRepository)(nil)
var _ rest.Repository = (*radioInfoRepository)(nil)
