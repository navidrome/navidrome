package persistence

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
)

type smartPlaylistJoinType int

const (
	smartPlaylistJoinNone            smartPlaylistJoinType = 0
	smartPlaylistJoinAlbumAnnotation smartPlaylistJoinType = 1 << iota
	smartPlaylistJoinArtistAnnotation
)

func (j smartPlaylistJoinType) has(other smartPlaylistJoinType) bool {
	return j&other != 0
}

type smartPlaylistField struct {
	expr     string
	order    string
	joinType smartPlaylistJoinType
}

type smartPlaylistCriteria struct {
	criteria.Criteria
	owner model.User
}

func newSmartPlaylistCriteria(c criteria.Criteria, opts ...func(*smartPlaylistCriteria)) smartPlaylistCriteria {
	cSQL := smartPlaylistCriteria{Criteria: c}
	for _, opt := range opts {
		opt(&cSQL)
	}
	return cSQL
}

func withSmartPlaylistOwner(owner model.User) func(*smartPlaylistCriteria) {
	return func(c *smartPlaylistCriteria) {
		c.owner = owner
	}
}

var smartPlaylistFields = map[string]smartPlaylistField{
	"title":                {expr: "media_file.title"},
	"album":                {expr: "media_file.album"},
	"hascoverart":          {expr: "media_file.has_cover_art"},
	"tracknumber":          {expr: "media_file.track_number"},
	"discnumber":           {expr: "media_file.disc_number"},
	"year":                 {expr: "media_file.year"},
	"date":                 {expr: "media_file.date"},
	"originalyear":         {expr: "media_file.original_year"},
	"originaldate":         {expr: "media_file.original_date"},
	"releaseyear":          {expr: "media_file.release_year"},
	"releasedate":          {expr: "media_file.release_date"},
	"size":                 {expr: "media_file.size"},
	"compilation":          {expr: "media_file.compilation"},
	"missing":              {expr: "media_file.missing"},
	"explicitstatus":       {expr: "media_file.explicit_status"},
	"dateadded":            {expr: "media_file.created_at"},
	"datemodified":         {expr: "media_file.updated_at"},
	"discsubtitle":         {expr: "media_file.disc_subtitle"},
	"comment":              {expr: "media_file.comment"},
	"lyrics":               {expr: "media_file.lyrics"},
	"sorttitle":            {expr: "media_file.sort_title"},
	"sortalbum":            {expr: "media_file.sort_album_name"},
	"sortartist":           {expr: "media_file.sort_artist_name"},
	"sortalbumartist":      {expr: "media_file.sort_album_artist_name"},
	"albumcomment":         {expr: "media_file.mbz_album_comment"},
	"catalognumber":        {expr: "media_file.catalog_num"},
	"filepath":             {expr: "media_file.path"},
	"filetype":             {expr: "media_file.suffix"},
	"codec":                {expr: "media_file.codec"},
	"duration":             {expr: "media_file.duration"},
	"bitrate":              {expr: "media_file.bit_rate"},
	"bitdepth":             {expr: "media_file.bit_depth"},
	"samplerate":           {expr: "media_file.sample_rate"},
	"bpm":                  {expr: "media_file.bpm"},
	"channels":             {expr: "media_file.channels"},
	"loved":                {expr: "COALESCE(annotation.starred, false)"},
	"dateloved":            {expr: "annotation.starred_at"},
	"lastplayed":           {expr: "annotation.play_date"},
	"daterated":            {expr: "annotation.rated_at"},
	"playcount":            {expr: "COALESCE(annotation.play_count, 0)"},
	"rating":               {expr: "COALESCE(annotation.rating, 0)"},
	"averagerating":        {expr: "media_file.average_rating"},
	"albumrating":          {expr: "COALESCE(album_annotation.rating, 0)", joinType: smartPlaylistJoinAlbumAnnotation},
	"albumloved":           {expr: "COALESCE(album_annotation.starred, false)", joinType: smartPlaylistJoinAlbumAnnotation},
	"albumplaycount":       {expr: "COALESCE(album_annotation.play_count, 0)", joinType: smartPlaylistJoinAlbumAnnotation},
	"albumlastplayed":      {expr: "album_annotation.play_date", joinType: smartPlaylistJoinAlbumAnnotation},
	"albumdateloved":       {expr: "album_annotation.starred_at", joinType: smartPlaylistJoinAlbumAnnotation},
	"albumdaterated":       {expr: "album_annotation.rated_at", joinType: smartPlaylistJoinAlbumAnnotation},
	"artistrating":         {expr: "COALESCE(artist_annotation.rating, 0)", joinType: smartPlaylistJoinArtistAnnotation},
	"artistloved":          {expr: "COALESCE(artist_annotation.starred, false)", joinType: smartPlaylistJoinArtistAnnotation},
	"artistplaycount":      {expr: "COALESCE(artist_annotation.play_count, 0)", joinType: smartPlaylistJoinArtistAnnotation},
	"artistlastplayed":     {expr: "artist_annotation.play_date", joinType: smartPlaylistJoinArtistAnnotation},
	"artistdateloved":      {expr: "artist_annotation.starred_at", joinType: smartPlaylistJoinArtistAnnotation},
	"artistdaterated":      {expr: "artist_annotation.rated_at", joinType: smartPlaylistJoinArtistAnnotation},
	"mbz_album_id":         {expr: "media_file.mbz_album_id"},
	"mbz_album_artist_id":  {expr: "media_file.mbz_album_artist_id"},
	"mbz_artist_id":        {expr: "media_file.mbz_artist_id"},
	"mbz_recording_id":     {expr: "media_file.mbz_recording_id"},
	"mbz_release_track_id": {expr: "media_file.mbz_release_track_id"},
	"mbz_release_group_id": {expr: "media_file.mbz_release_group_id"},
	"rgalbumgain":          {expr: "media_file.rg_album_gain"},
	"rgalbumpeak":          {expr: "media_file.rg_album_peak"},
	"rgtrackgain":          {expr: "media_file.rg_track_gain"},
	"rgtrackpeak":          {expr: "media_file.rg_track_peak"},
	"library_id":           {expr: "media_file.library_id"},
	"random":               {order: "random()"},
}

func (c smartPlaylistCriteria) Where() (squirrel.Sqlizer, error) {
	if c.Criteria.Expression == nil {
		return squirrel.Expr("1 = 1"), nil
	}
	return c.exprSQL(c.Criteria.Expression)
}

func (c smartPlaylistCriteria) exprSQL(expr criteria.Expression) (squirrel.Sqlizer, error) {
	switch e := expr.(type) {
	case criteria.All:
		and := squirrel.And{}
		for _, child := range e {
			cond, err := c.exprSQL(child)
			if err != nil {
				return nil, err
			}
			and = append(and, cond)
		}
		return and, nil
	case criteria.Any:
		or := squirrel.Or{}
		for _, child := range e {
			cond, err := c.exprSQL(child)
			if err != nil {
				return nil, err
			}
			or = append(or, cond)
		}
		return or, nil
	case criteria.Is:
		return mapExpr(e, func(fields map[string]any) squirrel.Sqlizer {
			return squirrel.Eq(fields)
		}, false)
	case criteria.IsNot:
		return isNotExpr(e)
	case criteria.Gt:
		return mapExpr(e, func(fields map[string]any) squirrel.Sqlizer {
			return squirrel.Gt(fields)
		}, false)
	case criteria.Lt:
		return mapExpr(e, func(fields map[string]any) squirrel.Sqlizer {
			return squirrel.Lt(fields)
		}, false)
	case criteria.Before:
		return mapExpr(e, func(fields map[string]any) squirrel.Sqlizer {
			return squirrel.Lt(fields)
		}, false)
	case criteria.After:
		return mapExpr(e, func(fields map[string]any) squirrel.Sqlizer {
			return squirrel.Gt(fields)
		}, false)
	case criteria.Contains:
		return likeExpr(e, "%%%v%%", false)
	case criteria.NotContains:
		return likeExpr(e, "%%%v%%", true)
	case criteria.StartsWith:
		return likeExpr(e, "%v%%", false)
	case criteria.EndsWith:
		return likeExpr(e, "%%%v", false)
	case criteria.InTheRange:
		return rangeExpr(e)
	case criteria.InTheLast:
		return periodExpr(e, false)
	case criteria.NotInTheLast:
		return periodExpr(e, true)
	case criteria.InPlaylist:
		return c.inList(e, false)
	case criteria.NotInPlaylist:
		return c.inList(e, true)
	case criteria.IsMissing:
		return missingExpr(e, true)
	case criteria.IsPresent:
		return missingExpr(e, false)
	default:
		return nil, fmt.Errorf("unknown criteria expression type %T", expr)
	}
}

func isNotExpr(values map[string]any) (squirrel.Sqlizer, error) {
	if _, value, info, ok := singleField(values); ok && (info.IsTag || info.IsRole) {
		return jsonExpr(info, squirrel.Eq{"value": value}, true), nil
	}
	fields, err := sqlFields(values)
	if err != nil {
		return nil, err
	}
	return squirrel.NotEq(fields), nil
}

func missingExpr(values map[string]any, checkAbsence bool) (squirrel.Sqlizer, error) {
	field, value, info, ok := singleField(values)
	if !ok {
		if len(values) != 1 {
			return nil, fmt.Errorf("invalid field in criteria: isMissing/isPresent requires exactly one field")
		}
		return nil, fmt.Errorf("invalid field in criteria: %s", field)
	}
	if !info.IsTag && !info.IsRole {
		return nil, fmt.Errorf("isMissing/isPresent operator is only supported for tag and role fields, got: %s", field)
	}

	b, ok := value.(bool)
	if !ok {
		return nil, fmt.Errorf("invalid boolean value for 'missing' expression: %s: %v", field, value)
	}
	negate := checkAbsence == b
	return jsonExpr(info, nil, negate), nil
}

func mapExpr(values map[string]any, makeCond func(map[string]any) squirrel.Sqlizer, negateJSON bool) (squirrel.Sqlizer, error) {
	if _, value, info, ok := singleField(values); ok && (info.IsTag || info.IsRole) {
		return jsonExpr(info, makeCond(map[string]any{"value": value}), negateJSON), nil
	}
	fields, err := sqlFields(values)
	if err != nil {
		return nil, err
	}
	return makeCond(fields), nil
}

func likeExpr(values map[string]any, pattern string, negate bool) (squirrel.Sqlizer, error) {
	if _, value, info, ok := singleField(values); ok && (info.IsTag || info.IsRole) {
		return jsonExpr(info, squirrel.Like{"value": fmt.Sprintf(pattern, value)}, negate), nil
	}
	fields, err := sqlFields(values)
	if err != nil {
		return nil, err
	}
	if negate {
		lk := squirrel.NotLike{}
		for field, value := range fields {
			lk[field] = fmt.Sprintf(pattern, value)
		}
		return lk, nil
	}
	lk := squirrel.Like{}
	for field, value := range fields {
		lk[field] = fmt.Sprintf(pattern, value)
	}
	return lk, nil
}

func rangeExpr(values map[string]any) (squirrel.Sqlizer, error) {
	fields, err := sqlFields(values)
	if err != nil {
		return nil, err
	}
	and := squirrel.And{}
	for field, value := range fields {
		s := reflect.ValueOf(value)
		if s.Kind() != reflect.Slice || s.Len() != 2 {
			return nil, fmt.Errorf("invalid range for 'in' operator: %s", value)
		}
		and = append(and,
			squirrel.GtOrEq{field: s.Index(0).Interface()},
			squirrel.LtOrEq{field: s.Index(1).Interface()},
		)
	}
	return and, nil
}

func periodExpr(values map[string]any, negate bool) (squirrel.Sqlizer, error) {
	fields, err := sqlFields(values)
	if err != nil {
		return nil, err
	}
	var field string
	var value any
	for f, v := range fields {
		field, value = f, v
		break
	}
	days, err := strconv.ParseInt(fmt.Sprintf("%v", value), 10, 64)
	if err != nil {
		return nil, err
	}
	firstDate := startOfPeriod(days, time.Now())
	if negate {
		return squirrel.Or{
			squirrel.Lt{field: firstDate},
			squirrel.Eq{field: nil},
		}, nil
	}
	return squirrel.Gt{field: firstDate}, nil
}

func startOfPeriod(numDays int64, from time.Time) string {
	return from.Add(time.Duration(-24*numDays) * time.Hour).Format("2006-01-02")
}

func (c smartPlaylistCriteria) inList(values map[string]any, negate bool) (squirrel.Sqlizer, error) {
	playlistID, ok := values["id"].(string)
	if !ok {
		return nil, errors.New("playlist id not given")
	}
	filters := squirrel.And{squirrel.Eq{"pl.playlist_id": playlistID}}
	if !c.owner.IsAdmin {
		if c.owner.ID == "" {
			filters = append(filters, squirrel.Eq{"playlist.public": 1})
		} else {
			filters = append(filters, squirrel.Or{
				squirrel.Eq{"playlist.public": 1},
				squirrel.Eq{"playlist.owner_id": c.owner.ID},
			})
		}
	}
	subQuery := squirrel.Select("media_file_id").
		From("playlist_tracks pl").
		LeftJoin("playlist on pl.playlist_id = playlist.id").
		Where(filters)
	subSQL, subArgs, err := subQuery.PlaceholderFormat(squirrel.Question).ToSql()
	if err != nil {
		return nil, err
	}
	if negate {
		return squirrel.Expr("media_file.id NOT IN ("+subSQL+")", subArgs...), nil
	}
	return squirrel.Expr("media_file.id IN ("+subSQL+")", subArgs...), nil
}

func jsonExpr(info criteria.FieldInfo, cond squirrel.Sqlizer, negate bool) squirrel.Sqlizer {
	if info.IsRole {
		return roleCond{role: info.Name(), cond: cond, not: negate}
	}
	return tagCond{tag: info.Name(), numeric: info.Numeric, cond: cond, not: negate}
}

type tagCond struct {
	tag     string
	numeric bool
	cond    squirrel.Sqlizer
	not     bool
}

func (e tagCond) ToSql() (string, []any, error) {
	var cond string
	var args []any
	var err error
	if e.cond != nil {
		cond, args, err = e.cond.ToSql()
		if e.numeric {
			cond = strings.ReplaceAll(cond, "value", "CAST(value AS REAL)")
		}
		cond = fmt.Sprintf("exists (select 1 from json_tree(media_file.tags, '$.%s') where key='value' and %s)", e.tag, cond)
	} else {
		cond = fmt.Sprintf("exists (select 1 from json_tree(media_file.tags, '$.%s') where key='value')", e.tag)
	}
	if e.not {
		cond = "not " + cond
	}
	return cond, args, err
}

type roleCond struct {
	role string
	cond squirrel.Sqlizer
	not  bool
}

func (e roleCond) ToSql() (string, []any, error) {
	var cond string
	var args []any
	var err error
	if e.cond != nil {
		cond, args, err = e.cond.ToSql()
		cond = fmt.Sprintf("exists (select 1 from json_tree(media_file.participants, '$.%s') where key='name' and %s)", e.role, cond)
	} else {
		cond = fmt.Sprintf("exists (select 1 from json_tree(media_file.participants, '$.%s') where key='name')", e.role)
	}
	if e.not {
		cond = "not " + cond
	}
	return cond, args, err
}

func singleField(values map[string]any) (string, any, criteria.FieldInfo, bool) {
	if len(values) != 1 {
		return "", nil, criteria.FieldInfo{}, false
	}
	for field, value := range values {
		info, ok := criteria.LookupField(field)
		return field, value, info, ok
	}
	return "", nil, criteria.FieldInfo{}, false
}

func sqlFields(values map[string]any) (map[string]any, error) {
	fields := make(map[string]any, len(values))
	for field, value := range values {
		info, ok := criteria.LookupField(field)
		if !ok {
			return nil, fmt.Errorf("invalid field in criteria: %s", field)
		}
		if info.IsTag || info.IsRole {
			return nil, fmt.Errorf("tag and role criteria must contain exactly one field: %s", field)
		}
		sqlField, ok := fieldExpr(info.Name())
		if !ok || sqlField == "" {
			return nil, fmt.Errorf("invalid field in criteria: %s", field)
		}
		fields[sqlField] = value
	}
	return fields, nil
}

func fieldExpr(name string) (string, bool) {
	field, ok := smartPlaylistFields[strings.ToLower(name)]
	return field.expr, ok
}

func fieldJoinType(name string) smartPlaylistJoinType {
	info, ok := criteria.LookupField(name)
	if !ok {
		return smartPlaylistJoinNone
	}
	field, ok := smartPlaylistFields[info.Name()]
	if !ok {
		return smartPlaylistJoinNone
	}
	return field.joinType
}

func (c smartPlaylistCriteria) ExpressionJoins() smartPlaylistJoinType {
	var joins smartPlaylistJoinType
	_ = criteria.Walk(c.Criteria.Expression, func(expr criteria.Expression) error {
		for field := range criteria.Fields(expr) {
			joins |= fieldJoinType(field)
		}
		return nil
	})
	return joins
}

func (c smartPlaylistCriteria) RequiredJoins() smartPlaylistJoinType {
	joins := c.ExpressionJoins()
	for _, name := range c.Criteria.SortFieldNames() {
		joins |= fieldJoinType(name)
	}
	return joins
}

func (c smartPlaylistCriteria) OrderBy() string {
	sortFields := c.Criteria.OrderByFields()
	parts := make([]string, 0, len(sortFields))
	for _, sf := range sortFields {
		mapped, ok := sortExpr(sf.Field)
		if !ok {
			continue
		}
		dir := "asc"
		if sf.Desc {
			dir = "desc"
		}
		parts = append(parts, mapped+" "+dir)
	}
	return strings.Join(parts, ", ")
}

func sortExpr(sortField string) (string, bool) {
	info, ok := criteria.LookupField(sortField)
	if !ok {
		return "", false
	}
	if field, ok := smartPlaylistFields[info.Name()]; ok && field.order != "" {
		return field.order, true
	}
	var mapped string
	switch {
	case info.IsTag:
		mapped = "COALESCE(json_extract(media_file.tags, '$." + info.Name() + "[0].value'), '')"
	case info.IsRole:
		mapped = "COALESCE(json_extract(media_file.participants, '$." + info.Name() + "[0].name'), '')"
	default:
		field, ok := smartPlaylistFields[info.Name()]
		if !ok || field.expr == "" {
			return "", false
		}
		mapped = field.expr
	}
	if info.Numeric {
		mapped = fmt.Sprintf("CAST(%s AS REAL)", mapped)
	}
	return mapped, true
}
