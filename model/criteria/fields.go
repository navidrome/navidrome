package criteria

import (
	"fmt"
	"strings"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/id"
)

var fieldMap = map[string]*mappedField{
	"title":           {field: "media_file.title"},
	"album":           {field: "media_file.album"},
	"artist":          {field: "media_file.artist"},
	"albumartist":     {field: "media_file.album_artist"},
	"hascoverart":     {field: "media_file.has_cover_art"},
	"tracknumber":     {field: "media_file.track_number"},
	"discnumber":      {field: "media_file.disc_number"},
	"year":            {field: "media_file.year"},
	"date":            {field: "media_file.date"},
	"originalyear":    {field: "media_file.original_year"},
	"originaldate":    {field: "media_file.original_date"},
	"releaseyear":     {field: "media_file.release_year"},
	"releasedate":     {field: "media_file.release_date"},
	"size":            {field: "media_file.size"},
	"compilation":     {field: "media_file.compilation"},
	"dateadded":       {field: "media_file.created_at"},
	"datemodified":    {field: "media_file.updated_at"},
	"discsubtitle":    {field: "media_file.disc_subtitle"},
	"comment":         {field: "media_file.comment"},
	"lyrics":          {field: "media_file.lyrics"},
	"sorttitle":       {field: "media_file.sort_title"},
	"sortalbum":       {field: "media_file.sort_album_name"},
	"sortartist":      {field: "media_file.sort_artist_name"},
	"sortalbumartist": {field: "media_file.sort_album_artist_name"},
	"albumtype":       {field: "media_file.mbz_album_type"},
	"albumcomment":    {field: "media_file.mbz_album_comment"},
	"catalognumber":   {field: "media_file.catalog_num"},
	"filepath":        {field: "media_file.path"},
	"filetype":        {field: "media_file.suffix"},
	"duration":        {field: "media_file.duration"},
	"bitrate":         {field: "media_file.bit_rate"},
	"bpm":             {field: "media_file.bpm"},
	"channels":        {field: "media_file.channels"},
	"loved":           {field: "COALESCE(annotation.starred, false)"},
	"dateloved":       {field: "annotation.starred_at"},
	"lastplayed":      {field: "annotation.play_date"},
	"playcount":       {field: "COALESCE(annotation.play_count, 0)"},
	"rating":          {field: "COALESCE(annotation.rating, 0)"},
	"random":          {field: "", order: "random()"},
	"genre":           {isTag: true},
}

type mappedField struct {
	field string
	order string
	isTag bool
}

func mapFields(expr map[string]interface{}) map[string]interface{} {
	m := make(map[string]interface{})
	for f, v := range expr {
		dbf := fieldMap[strings.ToLower(f)]
		if dbf == nil {
			log.Error("Invalid field in criteria", "field", f)
			continue
		}
		if dbf.field != "" {
			m[dbf.field] = v
			continue
		}
		if dbf.isTag {
			// BFR Should we validate tags that exist in the DB?
			// BFR Handle multi-valued tags
			k, v := firstKV(expr)
			tagID := id.NewTagID(k, v)
			m["tags.value"] = tagID
		}
	}
	return m
}

func firstKV(expr map[string]interface{}) (string, string) {
	for k, v := range expr {
		return k, fmt.Sprint(v)
	}
	return "", ""
}
