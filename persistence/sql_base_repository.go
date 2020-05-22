package persistence

import (
	"context"
	"fmt"
	"strings"
	"text/scanner"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/model/request"
	"github.com/google/uuid"
)

type sqlRepository struct {
	ctx          context.Context
	tableName    string
	ormer        orm.Ormer
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
			sort := toSnakeCase(options[0].Sort)
			if mapping, ok := r.sortMappings[sort]; ok {
				sort = mapping
			}
			if !strings.Contains(sort, "asc") && !strings.Contains(sort, "desc") {
				sort = sort + " asc"
			}
			if options[0].Order == "desc" {
				var s scanner.Scanner
				s.Init(strings.NewReader(sort))
				var newSort string
				for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
					switch s.TokenText() {
					case "asc":
						newSort += " " + "desc"
					case "desc":
						newSort += " " + "asc"
					default:
						newSort += " " + s.TokenText()
					}
				}
				sort = newSort
			}
			sq = sq.OrderBy(sort)
		}
	}
	return sq
}

func (r sqlRepository) applyFilters(sq SelectBuilder, options ...model.QueryOptions) SelectBuilder {
	if len(options) > 0 && options[0].Filters != nil {
		sq = sq.Where(options[0].Filters)
	}
	return sq
}

func (r sqlRepository) executeSQL(sq Sqlizer) (int64, error) {
	query, args, err := sq.ToSql()
	if err != nil {
		return 0, err
	}
	start := time.Now()
	var c int64
	res, err := r.ormer.Raw(query, args...).Exec()
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

// Note: Due to a bug in the QueryRow, this method does not map any embedded structs (ex: annotations)
// In this case, use the queryAll method and get the first item of the returned list
func (r sqlRepository) queryOne(sq Sqlizer, response interface{}) error {
	query, args, err := sq.ToSql()
	if err != nil {
		return err
	}
	start := time.Now()
	err = r.ormer.Raw(query, args...).QueryRow(response)
	if err == orm.ErrNoRows {
		r.logSQL(query, args, nil, 1, start)
		return model.ErrNotFound
	}
	r.logSQL(query, args, err, 1, start)
	return err
}

func (r sqlRepository) queryAll(sq Sqlizer, response interface{}) error {
	query, args, err := sq.ToSql()
	if err != nil {
		return err
	}
	start := time.Now()
	c, err := r.ormer.Raw(query, args...).QueryRows(response)
	if err == orm.ErrNoRows {
		r.logSQL(query, args, nil, c, start)
		return model.ErrNotFound
	}
	r.logSQL(query, args, nil, c, start)
	return err
}

func (r sqlRepository) exists(existsQuery SelectBuilder) (bool, error) {
	existsQuery = existsQuery.Columns("count(*) as exist").From(r.tableName)
	var res struct{ Exist int64 }
	err := r.queryOne(existsQuery, &res)
	return res.Exist > 0, err
}

func (r sqlRepository) count(countQuery SelectBuilder, options ...model.QueryOptions) (int64, error) {
	countQuery = countQuery.Columns("count(*) as count").From(r.tableName)
	countQuery = r.applyFilters(countQuery, options...)
	var res struct{ Count int64 }
	err := r.queryOne(countQuery, &res)
	return res.Count, err
}

func (r sqlRepository) put(id string, m interface{}) (newId string, err error) {
	values, _ := toSqlArgs(m)
	// Remove created_at from args and save it for later, if needed for insert
	createdAt := values["created_at"]
	delete(values, "created_at")
	if id != "" {
		update := Update(r.tableName).Where(Eq{"id": id}).SetMap(values)
		count, err := r.executeSQL(update)
		if err != nil {
			return "", err
		}
		if count > 0 {
			if _, ok := m.(model.AnnotatedModel); ok {
				err = r.updateAnnotations(id, m)
			}
			return id, err
		}
	}
	// If does not have an id OR could not update (new record with predefined id)
	if id == "" {
		rand, _ := uuid.NewRandom()
		id = rand.String()
		values["id"] = id
	}
	// It is a insert. if there was a created_at, add it back to args
	if createdAt != nil {
		values["created_at"] = createdAt
	}
	insert := Insert(r.tableName).SetMap(values)
	_, err = r.executeSQL(insert)
	return id, err
}

func (r sqlRepository) delete(cond Sqlizer) error {
	del := Delete(r.tableName).Where(cond)
	_, err := r.executeSQL(del)
	if err == orm.ErrNoRows {
		return model.ErrNotFound
	}
	return err
}

func (r sqlRepository) logSQL(sql string, args []interface{}, err error, rowsAffected int64, start time.Time) {
	elapsed := time.Since(start)
	var fmtArgs []string
	for i := range args {
		var f string
		switch a := args[i].(type) {
		case string:
			f = `'` + a + `'`
		default:
			f = fmt.Sprintf("%v", a)
		}
		fmtArgs = append(fmtArgs, f)
	}
	if err != nil {
		log.Error(r.ctx, "SQL: `"+sql+"`", "args", `[`+strings.Join(fmtArgs, ",")+`]`, "rowsAffected", rowsAffected, "elapsedTime", elapsed, err)
	} else {
		log.Trace(r.ctx, "SQL: `"+sql+"`", "args", `[`+strings.Join(fmtArgs, ",")+`]`, "rowsAffected", rowsAffected, "elapsedTime", elapsed)
	}
}
