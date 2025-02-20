package criteria

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/Masterminds/squirrel"
)

type (
	All squirrel.And
	And = All
)

func (all All) ToSql() (sql string, args []any, err error) {
	return squirrel.And(all).ToSql()
}

func (all All) MarshalJSON() ([]byte, error) {
	return marshalConjunction("all", all)
}

func (all All) ChildPlaylistIds() (ids []string) {
	return extractPlaylistIds(all)
}

type (
	Any squirrel.Or
	Or  = Any
)

func (any Any) ToSql() (sql string, args []any, err error) {
	return squirrel.Or(any).ToSql()
}

func (any Any) MarshalJSON() ([]byte, error) {
	return marshalConjunction("any", any)
}

func (any Any) ChildPlaylistIds() (ids []string) {
	return extractPlaylistIds(any)
}

type Is squirrel.Eq
type Eq = Is

func (is Is) ToSql() (sql string, args []any, err error) {
	if isRoleExpr(is) {
		return mapRoleExpr(is, false).ToSql()
	}
	if isTagExpr(is) {
		return mapTagExpr(is, false).ToSql()
	}
	return squirrel.Eq(mapFields(is)).ToSql()
}

func (is Is) MarshalJSON() ([]byte, error) {
	return marshalExpression("is", is)
}

type IsNot squirrel.NotEq

func (in IsNot) ToSql() (sql string, args []any, err error) {
	if isRoleExpr(in) {
		return mapRoleExpr(squirrel.Eq(in), true).ToSql()
	}
	if isTagExpr(in) {
		return mapTagExpr(squirrel.Eq(in), true).ToSql()
	}
	return squirrel.NotEq(mapFields(in)).ToSql()
}

func (in IsNot) MarshalJSON() ([]byte, error) {
	return marshalExpression("isNot", in)
}

type Gt squirrel.Gt

func (gt Gt) ToSql() (sql string, args []any, err error) {
	if isTagExpr(gt) {
		return mapTagExpr(gt, false).ToSql()
	}
	return squirrel.Gt(mapFields(gt)).ToSql()
}

func (gt Gt) MarshalJSON() ([]byte, error) {
	return marshalExpression("gt", gt)
}

type Lt squirrel.Lt

func (lt Lt) ToSql() (sql string, args []any, err error) {
	if isTagExpr(lt) {
		return mapTagExpr(squirrel.Lt(lt), false).ToSql()
	}
	return squirrel.Lt(mapFields(lt)).ToSql()
}

func (lt Lt) MarshalJSON() ([]byte, error) {
	return marshalExpression("lt", lt)
}

type Before squirrel.Lt

func (bf Before) ToSql() (sql string, args []any, err error) {
	return Lt(bf).ToSql()
}

func (bf Before) MarshalJSON() ([]byte, error) {
	return marshalExpression("before", bf)
}

type After Gt

func (af After) ToSql() (sql string, args []any, err error) {
	return Gt(af).ToSql()
}

func (af After) MarshalJSON() ([]byte, error) {
	return marshalExpression("after", af)
}

type Contains map[string]any

func (ct Contains) ToSql() (sql string, args []any, err error) {
	lk := squirrel.Like{}
	for f, v := range mapFields(ct) {
		lk[f] = fmt.Sprintf("%%%s%%", v)
	}
	if isRoleExpr(ct) {
		return mapRoleExpr(lk, false).ToSql()
	}
	if isTagExpr(ct) {
		return mapTagExpr(lk, false).ToSql()
	}
	return lk.ToSql()
}

func (ct Contains) MarshalJSON() ([]byte, error) {
	return marshalExpression("contains", ct)
}

type NotContains map[string]any

func (nct NotContains) ToSql() (sql string, args []any, err error) {
	lk := squirrel.NotLike{}
	for f, v := range mapFields(nct) {
		lk[f] = fmt.Sprintf("%%%s%%", v)
	}
	if isRoleExpr(nct) {
		return mapRoleExpr(squirrel.Like(lk), true).ToSql()
	}
	if isTagExpr(nct) {
		return mapTagExpr(squirrel.Like(lk), true).ToSql()
	}
	return lk.ToSql()
}

func (nct NotContains) MarshalJSON() ([]byte, error) {
	return marshalExpression("notContains", nct)
}

type StartsWith map[string]any

func (sw StartsWith) ToSql() (sql string, args []any, err error) {
	lk := squirrel.Like{}
	for f, v := range mapFields(sw) {
		lk[f] = fmt.Sprintf("%s%%", v)
	}
	if isRoleExpr(sw) {
		return mapRoleExpr(lk, false).ToSql()
	}
	if isTagExpr(sw) {
		return mapTagExpr(lk, false).ToSql()
	}
	return lk.ToSql()
}

func (sw StartsWith) MarshalJSON() ([]byte, error) {
	return marshalExpression("startsWith", sw)
}

type EndsWith map[string]any

func (sw EndsWith) ToSql() (sql string, args []any, err error) {
	lk := squirrel.Like{}
	for f, v := range mapFields(sw) {
		lk[f] = fmt.Sprintf("%%%s", v)
	}
	if isRoleExpr(sw) {
		return mapRoleExpr(lk, false).ToSql()
	}
	if isTagExpr(sw) {
		return mapTagExpr(lk, false).ToSql()
	}
	return lk.ToSql()
}

func (sw EndsWith) MarshalJSON() ([]byte, error) {
	return marshalExpression("endsWith", sw)
}

type InTheRange map[string]any

func (itr InTheRange) ToSql() (sql string, args []any, err error) {
	and := squirrel.And{}
	for f, v := range mapFields(itr) {
		s := reflect.ValueOf(v)
		if s.Kind() != reflect.Slice || s.Len() != 2 {
			return "", nil, fmt.Errorf("invalid range for 'in' operator: %s", v)
		}
		and = append(and,
			squirrel.GtOrEq{f: s.Index(0).Interface()},
			squirrel.LtOrEq{f: s.Index(1).Interface()},
		)
	}
	return and.ToSql()
}

func (itr InTheRange) MarshalJSON() ([]byte, error) {
	return marshalExpression("inTheRange", itr)
}

type InTheLast map[string]any

func (itl InTheLast) ToSql() (sql string, args []any, err error) {
	exp, err := inPeriod(itl, false)
	if err != nil {
		return "", nil, err
	}
	return exp.ToSql()
}

func (itl InTheLast) MarshalJSON() ([]byte, error) {
	return marshalExpression("inTheLast", itl)
}

type NotInTheLast map[string]any

func (nitl NotInTheLast) ToSql() (sql string, args []any, err error) {
	exp, err := inPeriod(nitl, true)
	if err != nil {
		return "", nil, err
	}
	return exp.ToSql()
}

func (nitl NotInTheLast) MarshalJSON() ([]byte, error) {
	return marshalExpression("notInTheLast", nitl)
}

func inPeriod(m map[string]any, negate bool) (Expression, error) {
	var field string
	var value any
	for f, v := range mapFields(m) {
		field, value = f, v
		break
	}
	str := fmt.Sprintf("%v", value)
	v, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return nil, err
	}
	firstDate := startOfPeriod(v, time.Now())

	if negate {
		return Or{
			squirrel.Lt{field: firstDate},
			squirrel.Eq{field: nil},
		}, nil
	}
	return squirrel.Gt{field: firstDate}, nil
}

func startOfPeriod(numDays int64, from time.Time) string {
	return from.Add(time.Duration(-24*numDays) * time.Hour).Format("2006-01-02")
}

type InPlaylist map[string]any

func (ipl InPlaylist) ToSql() (sql string, args []any, err error) {
	return inList(ipl, false)
}

func (ipl InPlaylist) MarshalJSON() ([]byte, error) {
	return marshalExpression("inPlaylist", ipl)
}

type NotInPlaylist map[string]any

func (ipl NotInPlaylist) ToSql() (sql string, args []any, err error) {
	return inList(ipl, true)
}

func (ipl NotInPlaylist) MarshalJSON() ([]byte, error) {
	return marshalExpression("notInPlaylist", ipl)
}

func inList(m map[string]any, negate bool) (sql string, args []any, err error) {
	var playlistid string
	var ok bool
	if playlistid, ok = m["id"].(string); !ok {
		return "", nil, errors.New("playlist id not given")
	}

	// Subquery to fetch all media files that are contained in given playlist
	// Only evaluate playlist if it is public
	subQuery := squirrel.Select("media_file_id").
		From("playlist_tracks pl").
		LeftJoin("playlist on pl.playlist_id = playlist.id").
		Where(squirrel.And{
			squirrel.Eq{"pl.playlist_id": playlistid},
			squirrel.Eq{"playlist.public": 1}})
	subQText, subQArgs, err := subQuery.PlaceholderFormat(squirrel.Question).ToSql()

	if err != nil {
		return "", nil, err
	}
	if negate {
		return "media_file.id NOT IN (" + subQText + ")", subQArgs, nil
	} else {
		return "media_file.id IN (" + subQText + ")", subQArgs, nil
	}
}

func extractPlaylistIds(inputRule any) (ids []string) {
	var id string
	var ok bool

	switch rule := inputRule.(type) {
	case Any:
		for _, rules := range rule {
			ids = append(ids, extractPlaylistIds(rules)...)
		}
	case All:
		for _, rules := range rule {
			ids = append(ids, extractPlaylistIds(rules)...)
		}
	case InPlaylist:
		if id, ok = rule["id"].(string); ok {
			ids = append(ids, id)
		}
	case NotInPlaylist:
		if id, ok = rule["id"].(string); ok {
			ids = append(ids, id)
		}
	}

	return
}
