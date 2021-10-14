package persistence

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
)

//{
// "combinator": "and",
// "rules": [
//   {"field": "loved", "operator": "is true"},
//   {"field": "lastPlayed", "operator": "in the last", "value": "90"}
// ],
// "order": "artist asc",
// "limit": 100
//}
type SmartPlaylist model.SmartPlaylist

func (sp SmartPlaylist) AddFilters(sql SelectBuilder) SelectBuilder {
	return sql.Where(RuleGroup(sp.RuleGroup)).OrderBy(sp.Order).Limit(uint64(sp.Limit))
}

type fieldDef struct {
	dbField  string
	ruleType reflect.Type
}

var fieldMap = map[string]*fieldDef{
	"title":           {"media_file.title", stringRuleType},
	"album":           {"media_file.album", stringRuleType},
	"artist":          {"media_file.artist", stringRuleType},
	"albumartist":     {"media_file.album_artist", stringRuleType},
	"albumartwork":    {"media_file.has_cover_art", stringRuleType},
	"tracknumber":     {"media_file.track_number", numberRuleType},
	"discnumber":      {"media_file.disc_number", numberRuleType},
	"year":            {"media_file.year", numberRuleType},
	"size":            {"media_file.size", numberRuleType},
	"compilation":     {"media_file.compilation", boolRuleType},
	"dateadded":       {"media_file.created_at", dateRuleType},
	"datemodified":    {"media_file.updated_at", dateRuleType},
	"discsubtitle":    {"media_file.disc_subtitle", stringRuleType},
	"comment":         {"media_file.comment", stringRuleType},
	"lyrics":          {"media_file.lyrics", stringRuleType},
	"sorttitle":       {"media_file.sort_title", stringRuleType},
	"sortalbum":       {"media_file.sort_album_name", stringRuleType},
	"sortartist":      {"media_file.sort_artist_name", stringRuleType},
	"sortalbumartist": {"media_file.sort_album_artist_name", stringRuleType},
	"albumtype":       {"media_file.mbz_album_type", stringRuleType},
	"albumcomment":    {"media_file.mbz_album_comment", stringRuleType},
	"catalognumber":   {"media_file.catalog_num", stringRuleType},
	"filepath":        {"media_file.path", stringRuleType},
	"filetype":        {"media_file.suffix", stringRuleType},
	"duration":        {"media_file.duration", numberRuleType},
	"bitrate":         {"media_file.bit_rate", numberRuleType},
	"bpm":             {"media_file.bpm", numberRuleType},
	"channels":        {"media_file.channels", numberRuleType},
	"genre":           {"genre.name", stringRuleType},
	"loved":           {"annotation.starred", boolRuleType},
	"lastplayed":      {"annotation.play_date", dateRuleType},
	"playcount":       {"annotation.play_count", numberRuleType},
	"rating":          {"annotation.rating", numberRuleType},
}

var stringRuleType = reflect.TypeOf(stringRule{})

type stringRule model.Rule

func (r stringRule) ToSql() (sql string, args []interface{}, err error) {
	var sq Sqlizer
	switch r.Operator {
	case "is":
		sq = Eq{r.Field: r.Value}
	case "is not":
		sq = NotEq{r.Field: r.Value}
	case "contains":
		sq = ILike{r.Field: fmt.Sprintf("%%%s%%", r.Value)}
	case "does not contains":
		sq = NotILike{r.Field: fmt.Sprintf("%%%s%%", r.Value)}
	case "begins with":
		sq = ILike{r.Field: fmt.Sprintf("%s%%", r.Value)}
	case "ends with":
		sq = ILike{r.Field: fmt.Sprintf("%%%s", r.Value)}
	default:
		return "", nil, errors.New("operator not supported: " + r.Operator)
	}
	return sq.ToSql()
}

var numberRuleType = reflect.TypeOf(numberRule{})

type numberRule model.Rule

func (r numberRule) ToSql() (sql string, args []interface{}, err error) {
	var sq Sqlizer
	switch r.Operator {
	case "is":
		sq = Eq{r.Field: r.Value}
	case "is not":
		sq = NotEq{r.Field: r.Value}
	case "is greater than":
		sq = Gt{r.Field: r.Value}
	case "is less than":
		sq = Lt{r.Field: r.Value}
	case "is in the range":
		s := reflect.ValueOf(r.Value)
		if s.Kind() != reflect.Slice || s.Len() != 2 {
			return "", nil, fmt.Errorf("invalid range for 'in' operator: %s", r.Value)
		}
		sq = And{
			GtOrEq{r.Field: s.Index(0).Interface()},
			LtOrEq{r.Field: s.Index(1).Interface()},
		}
	default:
		return "", nil, errors.New("operator not supported: " + r.Operator)
	}
	return sq.ToSql()
}

var dateRuleType = reflect.TypeOf(dateRule{})

type dateRule model.Rule

func (r dateRule) ToSql() (string, []interface{}, error) {
	var date time.Time
	var err error
	var sq Sqlizer
	switch r.Operator {
	case "is":
		date, err = r.parseDate(r.Value)
		sq = Eq{r.Field: date}
	case "is not":
		date, err = r.parseDate(r.Value)
		sq = NotEq{r.Field: date}
	case "is before":
		date, err = r.parseDate(r.Value)
		sq = Lt{r.Field: date}
	case "is after":
		date, err = r.parseDate(r.Value)
		sq = Gt{r.Field: date}
	case "is in the range":
		var dates []time.Time
		if dates, err = r.parseDates(); err == nil {
			sq = And{GtOrEq{r.Field: dates[0]}, LtOrEq{r.Field: dates[1]}}
		}
	case "in the last":
		sq, err = r.inTheLast(false)
	case "not in the last":
		sq, err = r.inTheLast(true)
	default:
		err = errors.New("operator not supported: " + r.Operator)
	}
	if err != nil {
		return "", nil, err
	}
	return sq.ToSql()
}

func (r dateRule) inTheLast(invert bool) (Sqlizer, error) {
	str := fmt.Sprintf("%v", r.Value)
	v, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return nil, err
	}
	period := time.Now().Add(time.Duration(-24*v) * time.Hour)
	if invert {
		return Lt{r.Field: period}, nil
	}
	return Gt{r.Field: period}, nil
}

func (r dateRule) parseDate(date interface{}) (time.Time, error) {
	input, ok := date.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("invalid date: %v", date)
	}
	d, err := time.Parse("2006-01-02", input)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date: %v", date)
	}
	return d, nil
}

func (r dateRule) parseDates() ([]time.Time, error) {
	input, ok := r.Value.([]string)
	if !ok {
		return nil, fmt.Errorf("invalid date range: %s", r.Value)
	}
	var dates []time.Time
	for _, s := range input {
		d, err := r.parseDate(s)
		if err != nil {
			return nil, fmt.Errorf("invalid date '%v' in range %v", s, input)
		}
		dates = append(dates, d)
	}
	if len(dates) != 2 {
		return nil, fmt.Errorf("not a valid date range: %s", r.Value)
	}
	return dates, nil
}

var boolRuleType = reflect.TypeOf(boolRule{})

type boolRule model.Rule

func (r boolRule) ToSql() (sql string, args []interface{}, err error) {
	var sq Sqlizer
	switch r.Operator {
	case "is true":
		sq = Eq{r.Field: true}
	case "is false":
		sq = Eq{r.Field: false}
	default:
		return "", nil, errors.New("operator not supported: " + r.Operator)
	}
	return sq.ToSql()
}

type RuleGroup model.RuleGroup

func (rg RuleGroup) ToSql() (sql string, args []interface{}, err error) {
	var sq []Sqlizer
	for _, r := range rg.Rules {
		switch rr := r.(type) {
		case model.Rule:
			sq = append(sq, rg.ruleToSqlizer(rr))
		case model.RuleGroup:
			sq = append(sq, RuleGroup(rr))
		}
	}
	var group Sqlizer
	if strings.ToLower(rg.Combinator) == "and" {
		group = And(sq)
	} else {
		group = Or(sq)
	}
	return group.ToSql()
}

type errorSqlizer string

func (e errorSqlizer) ToSql() (sql string, args []interface{}, err error) {
	return "", nil, errors.New(string(e))
}

func (rg RuleGroup) ruleToSqlizer(r model.Rule) Sqlizer {
	ruleDef := fieldMap[strings.ToLower(r.Field)]
	if ruleDef == nil {
		return errorSqlizer(fmt.Sprintf("invalid smart playlist field '%s'", r.Field))
	}
	r.Field = ruleDef.dbField
	r.Operator = strings.ToLower(r.Operator)
	switch ruleDef.ruleType {
	case stringRuleType:
		return stringRule(r)
	case numberRuleType:
		return numberRule(r)
	case boolRuleType:
		return boolRule(r)
	case dateRuleType:
		return dateRule(r)
	default:
		return errorSqlizer("invalid smart playlist rule type" + ruleDef.ruleType.String())
	}
}
