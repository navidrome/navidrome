package persistence

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/rest"
	"github.com/google/uuid"
)

type sqlRepository struct {
	ctx       context.Context
	tableName string
	ormer     orm.Ormer
}

const invalidUserId = "-1"

func userId(ctx context.Context) string {
	user := ctx.Value("user")
	if user == nil {
		return invalidUserId
	}
	usr := user.(*model.User)
	return usr.ID
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
			if options[0].Order == "desc" {
				sq = sq.OrderBy(toSnakeCase(options[0].Sort + " desc"))
			} else {
				sq = sq.OrderBy(toSnakeCase(options[0].Sort))
			}
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
	res, err := r.ormer.Raw(query, args...).Exec()
	c, _ := res.RowsAffected()
	r.logSQL(query, args, err, c, start)
	if err != nil {
		if err.Error() != "LastInsertId is not supported by this driver" {
			return 0, err
		}
	}
	return res.RowsAffected()
}

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
	if id != "" {
		update := Update(r.tableName).Where(Eq{"id": id}).SetMap(values)
		count, err := r.executeSQL(update)
		if err != nil {
			return "", err
		}
		if count > 0 {
			return id, nil
		}
	}
	// if does not have an id OR could not update (new record with predefined id)
	if id == "" {
		rand, _ := uuid.NewRandom()
		id = rand.String()
		values["id"] = id
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
	lapsed := time.Since(start)
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
		log.Error(r.ctx, "SQL: `"+sql+"`", "args", `[`+strings.Join(fmtArgs, ",")+`]`, "rowsAffected", rowsAffected, "lapsedTime", lapsed, err)
	} else {
		log.Trace(r.ctx, "SQL: `"+sql+"`", "args", `[`+strings.Join(fmtArgs, ",")+`]`, "rowsAffected", rowsAffected, "lapsedTime", lapsed)
	}
}

func (r sqlRepository) parseRestOptions(options ...rest.QueryOptions) model.QueryOptions {
	qo := model.QueryOptions{}
	if len(options) > 0 {
		qo.Sort = options[0].Sort
		qo.Order = options[0].Order
		qo.Max = options[0].Max
		qo.Offset = options[0].Offset
		if len(options[0].Filters) > 0 {
			filters := And{}
			for f, v := range options[0].Filters {
				filters = append(filters, Like{f: fmt.Sprintf("%s%%", v)})
			}
			qo.Filters = filters
		}
	}
	return qo
}
