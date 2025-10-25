package criteria

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
)

var fieldMap = map[string]*mappedField{
	"title":                {field: "media_file.title"},
	"album":                {field: "media_file.album"},
	"hascoverart":          {field: "media_file.has_cover_art"},
	"tracknumber":          {field: "media_file.track_number"},
	"discnumber":           {field: "media_file.disc_number"},
	"year":                 {field: "media_file.year"},
	"date":                 {field: "media_file.date", alias: "recordingdate"},
	"originalyear":         {field: "media_file.original_year"},
	"originaldate":         {field: "media_file.original_date"},
	"releaseyear":          {field: "media_file.release_year"},
	"releasedate":          {field: "media_file.release_date"},
	"size":                 {field: "media_file.size"},
	"compilation":          {field: "media_file.compilation"},
	"dateadded":            {field: "media_file.created_at"},
	"datemodified":         {field: "media_file.updated_at"},
	"discsubtitle":         {field: "media_file.disc_subtitle"},
	"comment":              {field: "media_file.comment"},
	"lyrics":               {field: "media_file.lyrics"},
	"sorttitle":            {field: "media_file.sort_title"},
	"sortalbum":            {field: "media_file.sort_album_name"},
	"sortartist":           {field: "media_file.sort_artist_name"},
	"sortalbumartist":      {field: "media_file.sort_album_artist_name"},
	"albumtype":            {field: "media_file.mbz_album_type", alias: "releasetype"},
	"albumcomment":         {field: "media_file.mbz_album_comment"},
	"catalognumber":        {field: "media_file.catalog_num"},
	"filepath":             {field: "media_file.path"},
	"filetype":             {field: "media_file.suffix"},
	"duration":             {field: "media_file.duration"},
	"bitrate":              {field: "media_file.bit_rate"},
	"bitdepth":             {field: "media_file.bit_depth"},
	"bpm":                  {field: "media_file.bpm"},
	"channels":             {field: "media_file.channels"},
	"loved":                {field: "COALESCE(annotation.starred, false)"},
	"dateloved":            {field: "annotation.starred_at"},
	"lastplayed":           {field: "annotation.play_date"},
	"playcount":            {field: "COALESCE(annotation.play_count, 0)"},
	"rating":               {field: "COALESCE(annotation.rating, 0)"},
	"mbz_album_id":         {field: "media_file.mbz_album_id"},
	"mbz_album_artist_id":  {field: "media_file.mbz_album_artist_id"},
	"mbz_artist_id":        {field: "media_file.mbz_artist_id"},
	"mbz_recording_id":     {field: "media_file.mbz_recording_id"},
	"mbz_release_track_id": {field: "media_file.mbz_release_track_id"},
	"mbz_release_group_id": {field: "media_file.mbz_release_group_id"},
	"library_id":           {field: "media_file.library_id", numeric: true},

	// special fields
	"random": {field: "", order: "random()"}, // pseudo-field for random sorting
	"value":  {field: "value"},               // pseudo-field for tag and roles values
}

type mappedField struct {
	field   string
	order   string
	isRole  bool   // true if the field is a role (e.g. "artist", "composer", "conductor", etc.)
	isTag   bool   // true if the field is a tag imported from the file metadata
	alias   string // name from `mappings.yml` that may differ from the name used in the smart playlist
	numeric bool   // true if the field/tag should be treated as numeric
}

func mapFields(expr map[string]any) map[string]any {
	m := make(map[string]any)
	for f, v := range expr {
		if dbf := fieldMap[strings.ToLower(f)]; dbf != nil && dbf.field != "" {
			m[dbf.field] = v
		} else {
			log.Error("Invalid field in criteria", "field", f)
		}
	}
	return m
}

// mapExpr maps a normal field expression to a specific type of expression (tag or role).
// This is required because tags are handled differently than other fields,
// as they are stored as a JSON column in the database.
func mapExpr(expr squirrel.Sqlizer, negate bool, exprFunc func(string, squirrel.Sqlizer, bool) squirrel.Sqlizer) squirrel.Sqlizer {
	rv := reflect.ValueOf(expr)
	if rv.Kind() != reflect.Map || rv.Type().Key().Kind() != reflect.String {
		log.Fatal(fmt.Sprintf("expr is not a map-based operator: %T", expr))
	}

	// Extract into a generic map
	var k string
	m := make(map[string]any, rv.Len())
	for _, key := range rv.MapKeys() {
		// Save the key to build the expression, and use the provided keyName as the key
		k = key.String()
		m["value"] = rv.MapIndex(key).Interface()
		break // only one key is expected (and supported)
	}

	// Clear the original map
	for _, key := range rv.MapKeys() {
		rv.SetMapIndex(key, reflect.Value{})
	}

	// Write the updated map back into the original variable
	for key, val := range m {
		rv.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(val))
	}

	return exprFunc(k, expr, negate)
}

// mapTagExpr maps a normal field expression to a tag expression.
func mapTagExpr(expr squirrel.Sqlizer, negate bool) squirrel.Sqlizer {
	return mapExpr(expr, negate, tagExpr)
}

// mapRoleExpr maps a normal field expression to an artist role expression.
func mapRoleExpr(expr squirrel.Sqlizer, negate bool) squirrel.Sqlizer {
	return mapExpr(expr, negate, roleExpr)
}

func isTagExpr(expr map[string]any) bool {
	for f := range expr {
		if f2, ok := fieldMap[strings.ToLower(f)]; ok && f2.isTag {
			return true
		}
	}
	return false
}

func isRoleExpr(expr map[string]any) bool {
	for f := range expr {
		if f2, ok := fieldMap[strings.ToLower(f)]; ok && f2.isRole {
			return true
		}
	}
	return false
}

func tagExpr(tag string, cond squirrel.Sqlizer, negate bool) squirrel.Sqlizer {
	return tagCond{tag: tag, cond: cond, not: negate}
}

type tagCond struct {
	tag  string
	cond squirrel.Sqlizer
	not  bool
}

func (e tagCond) ToSql() (string, []any, error) {
	cond, args, err := e.cond.ToSql()

	// Check if this tag is marked as numeric in the fieldMap
	if fm, ok := fieldMap[e.tag]; ok && fm.numeric {
		cond = strings.ReplaceAll(cond, "value", "CAST(value AS REAL)")
	}

	cond = fmt.Sprintf("exists (select 1 from json_tree(tags, '$.%s') where key='value' and %s)",
		e.tag, cond)
	if e.not {
		cond = "not " + cond
	}
	return cond, args, err
}

func roleExpr(role string, cond squirrel.Sqlizer, negate bool) squirrel.Sqlizer {
	return roleCond{role: role, cond: cond, not: negate}
}

type roleCond struct {
	role string
	cond squirrel.Sqlizer
	not  bool
}

func (e roleCond) ToSql() (string, []any, error) {
	cond, args, err := e.cond.ToSql()
	cond = fmt.Sprintf(`exists (select 1 from json_tree(participants, '$.%s') where key='name' and %s)`,
		e.role, cond)
	if e.not {
		cond = "not " + cond
	}
	return cond, args, err
}

// AddRoles adds roles to the field map. This is used to add all artist roles to the field map, so they can be used in
// smart playlists. If a role already exists in the field map, it is ignored, so calls to this function are idempotent.
func AddRoles(roles []string) {
	for _, role := range roles {
		name := strings.ToLower(role)
		if _, ok := fieldMap[name]; ok {
			continue
		}
		fieldMap[name] = &mappedField{field: name, isRole: true}
	}
}

// AddTagNames adds tag names to the field map. This is used to add all tags mapped in the `mappings.yml`
// file to the field map, so they can be used in smart playlists.
// If a tag name already exists in the field map, it is ignored, so calls to this function are idempotent.
func AddTagNames(tagNames []string) {
	for _, name := range tagNames {
		name := strings.ToLower(name)
		if _, ok := fieldMap[name]; ok {
			continue
		}
		for _, fm := range fieldMap {
			if fm.alias == name {
				fieldMap[name] = fm
				break
			}
		}
		if _, ok := fieldMap[name]; !ok {
			fieldMap[name] = &mappedField{field: name, isTag: true}
		}
	}
}

// AddNumericTags marks the given tag names as numeric so they can be cast
// when used in comparisons or sorting.
func AddNumericTags(tagNames []string) {
	for _, name := range tagNames {
		name := strings.ToLower(name)
		if fm, ok := fieldMap[name]; ok {
			fm.numeric = true
		} else {
			fieldMap[name] = &mappedField{field: name, isTag: true, numeric: true}
		}
	}
}
