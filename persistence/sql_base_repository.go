package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils/hasher"
	"github.com/pocketbase/dbx"
)

// sqlRepository is the base repository for all SQL repositories. It provides common functions to interact with the DB.
// When creating a new repository using this base, you must:
//
//   - Embed this struct.
//   - Set ctx and db fields. ctx should be the context passed to the constructor method, usually obtained from the request
//   - Call registerModel with the model instance and any possible filters.
//   - If the model has a different table name than the default (lowercase of the model name), it should be set manually
//     using the tableName field.
//   - Sort mappings should be set in the sortMappings field. If the sort field is not in the map, it will be used as is.
//
// All fields in filters and sortMappings must be in snake_case. Only sorts and filters based on real field names or
// defined in the mappings will be allowed.
type sqlRepository struct {
	ctx                context.Context
	tableName          string
	db                 dbx.Builder
	sortMappings       map[string]string
	filterMappings     map[string]filterFunc
	isFieldWhiteListed fieldWhiteListedFunc
}

const invalidUserId = "-1"

func userId(ctx context.Context) string {
	if user, ok := request.UserFrom(ctx); !ok {
		return invalidUserId
	} else {
		return user.ID
	}
}

func loggedUser(ctx context.Context) *model.User {
	if user, ok := request.UserFrom(ctx); !ok {
		return &model.User{}
	} else {
		return &user
	}
}

func (r *sqlRepository) registerModel(instance any, filters map[string]filterFunc) {
	if r.tableName == "" {
		r.tableName = strings.TrimPrefix(reflect.TypeOf(instance).String(), "*model.")
		r.tableName = toSnakeCase(r.tableName)
	}
	r.tableName = strings.ToLower(r.tableName)
	r.isFieldWhiteListed = registerModelWhiteList(instance)
	r.filterMappings = filters
}

func (r sqlRepository) getTableName() string {
	return r.tableName
}

func (r sqlRepository) newSelect(options ...model.QueryOptions) SelectBuilder {
	sq := Select().From(r.tableName)
	sq = r.applyOptions(sq, options...)
	sq = r.applyFilters(sq, options...)
	return sq
}

func (r sqlRepository) applyOptions(sq SelectBuilder, options ...model.QueryOptions) SelectBuilder {
	if len(options) > 0 {
		if options[0].Max > 0 {
			sq = sq.Limit(uint64(options[0].Max))
		}
		if options[0].Offset > 0 {
			sq = sq.Offset(uint64(options[0].Offset))
		}
		if options[0].Sort != "" {
			sq = sq.OrderBy(r.buildSortOrder(options[0].Sort, options[0].Order))
		}
	}
	return sq
}

// TODO Change all sortMappings to have a consistent case
func (r sqlRepository) sortMapping(sort string) string {
	if mapping, ok := r.sortMappings[sort]; ok {
		return mapping
	}
	if mapping, ok := r.sortMappings[toCamelCase(sort)]; ok {
		return mapping
	}
	sort = toSnakeCase(sort)
	if mapping, ok := r.sortMappings[sort]; ok {
		return mapping
	}
	return sort
}

func (r sqlRepository) buildSortOrder(sort, order string) string {
	sort = r.sortMapping(sort)
	order = strings.ToLower(strings.TrimSpace(order))
	var reverseOrder string
	if order == "desc" {
		reverseOrder = "asc"
	} else {
		order = "asc"
		reverseOrder = "desc"
	}

	var newSort []string
	parts := strings.FieldsFunc(sort, splitFunc(','))
	for _, p := range parts {
		f := strings.FieldsFunc(p, splitFunc(' '))
		newField := []string{f[0]}
		if len(f) == 1 {
			newField = append(newField, order)
		} else {
			if f[1] == "asc" {
				newField = append(newField, order)
			} else {
				newField = append(newField, reverseOrder)
			}
		}
		newSort = append(newSort, strings.Join(newField, " "))
	}
	return strings.Join(newSort, ", ")
}

func splitFunc(delimiter rune) func(c rune) bool {
	open := 0
	return func(c rune) bool {
		if c == '(' {
			open++
			return false
		}
		if open > 0 {
			if c == ')' {
				open--
			}
			return false
		}
		return c == delimiter
	}
}

func (r sqlRepository) applyFilters(sq SelectBuilder, options ...model.QueryOptions) SelectBuilder {
	if len(options) > 0 && options[0].Filters != nil {
		sq = sq.Where(options[0].Filters)
	}
	return sq
}

func (r sqlRepository) seedKey() string {
	return r.tableName + userId(r.ctx)
}

func (r sqlRepository) resetSeededRandom(options []model.QueryOptions) {
	if len(options) == 0 || options[0].Sort != "random" {
		return
	}
	options[0].Sort = fmt.Sprintf("SEEDEDRAND('%s', %s.id)", r.seedKey(), r.tableName)
	if options[0].Seed != "" {
		hasher.SetSeed(r.seedKey(), options[0].Seed)
		return
	}
	if options[0].Offset == 0 {
		hasher.Reseed(r.seedKey())
	}
}

func (r sqlRepository) executeSQL(sq Sqlizer) (int64, error) {
	query, args, err := r.toSQL(sq)
	if err != nil {
		return 0, err
	}
	start := time.Now()
	var c int64
	res, err := r.db.NewQuery(query).Bind(args).WithContext(r.ctx).Execute()
	if res != nil {
		c, _ = res.RowsAffected()
	}
	r.logSQL(query, args, err, c, start)
	if err != nil {
		if err.Error() != "LastInsertId is not supported by this driver" {
			return 0, err
		}
	}
	return res.RowsAffected()
}

func (r sqlRepository) toSQL(sq Sqlizer) (string, dbx.Params, error) {
	query, args, err := sq.ToSql()
	if err != nil {
		return "", nil, err
	}
	// Replace query placeholders with named params
	params := dbx.Params{}
	for i, arg := range args {
		p := fmt.Sprintf("p%d", i)
		query = strings.Replace(query, "?", "{:"+p+"}", 1)
		params[p] = arg
	}
	return query, params, nil
}

func (r sqlRepository) queryOne(sq Sqlizer, response interface{}) error {
	query, args, err := r.toSQL(sq)
	if err != nil {
		return err
	}
	start := time.Now()
	err = r.db.NewQuery(query).Bind(args).WithContext(r.ctx).One(response)
	if errors.Is(err, sql.ErrNoRows) {
		r.logSQL(query, args, nil, 0, start)
		return model.ErrNotFound
	}
	r.logSQL(query, args, err, 1, start)
	return err
}

func (r sqlRepository) queryAll(sq SelectBuilder, response interface{}, options ...model.QueryOptions) error {
	if len(options) > 0 && options[0].Offset > 0 {
		sq = r.optimizePagination(sq, options[0])
	}
	query, args, err := r.toSQL(sq)
	if err != nil {
		return err
	}
	start := time.Now()
	err = r.db.NewQuery(query).Bind(args).WithContext(r.ctx).All(response)
	if errors.Is(err, sql.ErrNoRows) {
		r.logSQL(query, args, nil, -1, start)
		return model.ErrNotFound
	}
	r.logSQL(query, args, err, int64(reflect.ValueOf(response).Elem().Len()), start)
	return err
}

// queryAllSlice is a helper function to query a single column and return the result in a slice
func (r sqlRepository) queryAllSlice(sq SelectBuilder, response interface{}) error {
	query, args, err := r.toSQL(sq)
	if err != nil {
		return err
	}
	start := time.Now()
	err = r.db.NewQuery(query).Bind(args).WithContext(r.ctx).Column(response)
	if errors.Is(err, sql.ErrNoRows) {
		r.logSQL(query, args, nil, -1, start)
		return model.ErrNotFound
	}
	r.logSQL(query, args, err, int64(reflect.ValueOf(response).Elem().Len()), start)
	return err
}

// optimizePagination uses a less inefficient pagination, by not using OFFSET.
// See https://gist.github.com/ssokolow/262503
func (r sqlRepository) optimizePagination(sq SelectBuilder, options model.QueryOptions) SelectBuilder {
	if options.Offset > conf.Server.DevOffsetOptimize {
		sq = sq.RemoveOffset()
		oidSq := sq.RemoveColumns().Columns(r.tableName + ".oid")
		oidSq = oidSq.Limit(uint64(options.Offset))
		oidSql, args, _ := oidSq.ToSql()
		sq = sq.Where(r.tableName+".oid not in ("+oidSql+")", args...)
	}
	return sq
}

func (r sqlRepository) exists(existsQuery SelectBuilder) (bool, error) {
	existsQuery = existsQuery.Columns("count(*) as exist").From(r.tableName)
	var res struct{ Exist int64 }
	err := r.queryOne(existsQuery, &res)
	return res.Exist > 0, err
}

func (r sqlRepository) count(countQuery SelectBuilder, options ...model.QueryOptions) (int64, error) {
	countQuery = countQuery.
		RemoveColumns().Columns("count(distinct " + r.tableName + ".id) as count").
		RemoveOffset().RemoveLimit().
		From(r.tableName)
	countQuery = r.applyFilters(countQuery, options...)
	var res struct{ Count int64 }
	err := r.queryOne(countQuery, &res)
	return res.Count, err
}

func (r sqlRepository) put(id string, m interface{}, colsToUpdate ...string) (newId string, err error) {
	values, err := toSQLArgs(m)
	if err != nil {
		return "", fmt.Errorf("error preparing values to write to DB: %w", err)
	}
	// If there's an ID, try to update first
	if id != "" {
		updateValues := map[string]interface{}{}

		// This is a map of the columns that need to be updated, if specified
		c2upd := map[string]struct{}{}
		for _, c := range colsToUpdate {
			c2upd[toSnakeCase(c)] = struct{}{}
		}
		for k, v := range values {
			if _, found := c2upd[k]; len(c2upd) == 0 || found {
				updateValues[k] = v
			}
		}

		delete(updateValues, "created_at")
		update := Update(r.tableName).Where(Eq{"id": id}).SetMap(updateValues)
		count, err := r.executeSQL(update)
		if err != nil {
			return "", err
		}
		if count > 0 {
			return id, nil
		}
	}
	// If it does not have an ID OR the ID was not found (when it is a new record with predefined id)
	if id == "" {
		id = uuid.NewString()
		values["id"] = id
	}
	insert := Insert(r.tableName).SetMap(values)
	_, err = r.executeSQL(insert)
	return id, err
}

func (r sqlRepository) delete(cond Sqlizer) error {
	del := Delete(r.tableName).Where(cond)
	_, err := r.executeSQL(del)
	if errors.Is(err, sql.ErrNoRows) {
		return model.ErrNotFound
	}
	return err
}

func (r sqlRepository) logSQL(sql string, args dbx.Params, err error, rowsAffected int64, start time.Time) {
	elapsed := time.Since(start)
	//var fmtArgs []string
	//for name, val := range args {
	//	var f string
	//	switch a := args[val].(type) {
	//	case string:
	//		f = `'` + a + `'`
	//	default:
	//		f = fmt.Sprintf("%v", a)
	//	}
	//	fmtArgs = append(fmtArgs, f)
	//}
	if err != nil {
		log.Error(r.ctx, "SQL: `"+sql+"`", "args", args, "rowsAffected", rowsAffected, "elapsedTime", elapsed, err)
	} else {
		log.Trace(r.ctx, "SQL: `"+sql+"`", "args", args, "rowsAffected", rowsAffected, "elapsedTime", elapsed)
	}
}
