package persistence

import (
	"context"
	"crypto/md5"
	"database/sql"
	"errors"
	"fmt"
	"iter"
	"reflect"
	"regexp"
	"strings"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	id2 "github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils/hasher"
	"github.com/navidrome/navidrome/utils/slice"
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
//   - Sort mappings must be set with setSortMappings method. If a sort field is not in the map, it will be used as the name of the column.
//
// All fields in filters and sortMappings must be in snake_case. Only sorts and filters based on real field names or
// defined in the mappings will be allowed.
type sqlRepository struct {
	ctx       context.Context
	tableName string
	db        dbx.Builder

	// Do not set these fields manually, they are set by the registerModel method
	filterMappings     map[string]filterFunc
	isFieldWhiteListed fieldWhiteListedFunc
	// Do not set this field manually, it is set by the setSortMappings method
	sortMappings map[string]string
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

// setSortMappings sets the mappings for the sort fields. If the sort field is not in the map, it will be used as is.
//
// If PreferSortTags is enabled, it will map the order fields to the corresponding sort expression,
// which gives precedence to sort tags.
// Ex: order_title => (coalesce(nullif(sort_title,”),order_title) collate nocase)
// To avoid performance issues, indexes should be created for these sort expressions
func (r *sqlRepository) setSortMappings(mappings map[string]string, tableName ...string) {
	tn := r.tableName
	if len(tableName) > 0 {
		tn = tableName[0]
	}
	if conf.Server.PreferSortTags {
		for k, v := range mappings {
			v = mapSortOrder(tn, v)
			mappings[k] = v
		}
	}
	r.sortMappings = mappings
}

func (r sqlRepository) newSelect(options ...model.QueryOptions) SelectBuilder {
	sq := Select().From(r.tableName)
	if len(options) > 0 {
		r.resetSeededRandom(options)
		sq = r.applyOptions(sq, options...)
		sq = r.applyFilters(sq, options...)
	}
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

	parts := strings.FieldsFunc(sort, splitFunc(','))
	newSort := make([]string, 0, len(parts))
	for _, p := range parts {
		f := strings.FieldsFunc(p, splitFunc(' '))
		newField := make([]string, 1, len(f))
		newField[0] = f[0]
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
	// Seed keys must be all lowercase, or else SQLite3 will encode it, making it not match the seed
	// used in the query. Hashing the user ID and converting it to a hex string will do the trick
	userIDHash := md5.Sum([]byte(userId(r.ctx)))
	return fmt.Sprintf("%s|%x", r.tableName, userIDHash)
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
	return c, err
}

var placeholderRegex = regexp.MustCompile(`\?`)

func (r sqlRepository) toSQL(sq Sqlizer) (string, dbx.Params, error) {
	query, args, err := sq.ToSql()
	if err != nil {
		return "", nil, err
	}
	// Replace query placeholders with named params
	params := make(dbx.Params, len(args))
	counter := 0
	result := placeholderRegex.ReplaceAllStringFunc(query, func(_ string) string {
		p := fmt.Sprintf("p%d", counter)
		params[p] = args[counter]
		counter++
		return "{:" + p + "}"
	})
	return result, params, nil
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

// queryWithStableResults is a helper function to execute a query and return an iterator that will yield its results
// from a cursor, guaranteeing that the results will be stable, even if the underlying data changes.
func queryWithStableResults[T any](r sqlRepository, sq SelectBuilder, options ...model.QueryOptions) (iter.Seq2[T, error], error) {
	if len(options) > 0 && options[0].Offset > 0 {
		sq = r.optimizePagination(sq, options[0])
	}
	query, args, err := r.toSQL(sq)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	rows, err := r.db.NewQuery(query).Bind(args).WithContext(r.ctx).Rows()
	r.logSQL(query, args, err, -1, start)
	if err != nil {
		return nil, err
	}
	return func(yield func(T, error) bool) {
		defer rows.Close()
		for rows.Next() {
			var row T
			err := rows.ScanStruct(&row)
			if !yield(row, err) || err != nil {
				return
			}
		}
		if err := rows.Err(); err != nil {
			var empty T
			yield(empty, err)
		}
	}, nil
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
		rowidSq := sq.RemoveColumns().Columns(r.tableName + ".rowid")
		rowidSq = rowidSq.Limit(uint64(options.Offset))
		rowidSql, args, _ := rowidSq.ToSql()
		sq = sq.Where(r.tableName+".rowid not in ("+rowidSql+")", args...)
	}
	return sq
}

func (r sqlRepository) exists(cond Sqlizer) (bool, error) {
	existsQuery := Select("count(*) as exist").From(r.tableName).Where(cond)
	var res struct{ Exist int64 }
	err := r.queryOne(existsQuery, &res)
	return res.Exist > 0, err
}

func (r sqlRepository) count(countQuery SelectBuilder, options ...model.QueryOptions) (int64, error) {
	countQuery = countQuery.
		RemoveColumns().Columns("count(distinct " + r.tableName + ".id) as count").
		RemoveOffset().RemoveLimit().
		OrderBy(r.tableName + ".id"). // To remove any ORDER BY clause that could slow down the query
		From(r.tableName)
	countQuery = r.applyFilters(countQuery, options...)
	var res struct{ Count int64 }
	err := r.queryOne(countQuery, &res)
	return res.Count, err
}

func (r sqlRepository) putByMatch(filter Sqlizer, id string, m interface{}, colsToUpdate ...string) (string, error) {
	if id != "" {
		return r.put(id, m, colsToUpdate...)
	}
	existsQuery := r.newSelect().Columns("id").From(r.tableName).Where(filter)

	var res struct{ ID string }
	err := r.queryOne(existsQuery, &res)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		return "", err
	}
	return r.put(res.ID, m, colsToUpdate...)
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
		c2upd := slice.ToMap(colsToUpdate, func(s string) (string, struct{}) {
			return toSnakeCase(s), struct{}{}
		})
		for k, v := range values {
			if _, found := c2upd[k]; len(c2upd) == 0 || found {
				updateValues[k] = v
			}
		}

		updateValues["id"] = id
		delete(updateValues, "created_at")
		// To avoid updating the media_file birth_time on each scan. Not the best solution, but it works for now
		// TODO move to mediafile_repository when each repo has its own upsert method
		delete(updateValues, "birth_time")
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
		id = id2.NewRandom()
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
	if err == nil || errors.Is(err, context.Canceled) {
		log.Trace(r.ctx, "SQL: `"+sql+"`", "args", args, "rowsAffected", rowsAffected, "elapsedTime", elapsed, err)
	} else {
		log.Error(r.ctx, "SQL: `"+sql+"`", "args", args, "rowsAffected", rowsAffected, "elapsedTime", elapsed, err)
	}
}
