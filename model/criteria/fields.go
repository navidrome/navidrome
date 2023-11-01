package criteria

import (
	"strings"

	"github.com/navidrome/navidrome/log"
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
	"genre":           {field: "COALESCE(genre.name, '')"},
	"loved":           {field: "COALESCE(annotation.starred, false)"},
	"dateloved":       {field: "annotation.starred_at"},
	"lastplayed":      {field: "annotation.play_date"},
	"playcount":       {field: "COALESCE(annotation.play_count, 0)"},
	"rating":          {field: "COALESCE(annotation.rating, 0)"},
	"random":          {field: "", order: "random()"},
}

type mappedField struct {
	field string
	order string
}

func mapFields(expr map[string]interface{}) map[string]interface{} {
	m := make(map[string]interface{})
	for f, v := range expr {
		if dbf := fieldMap[strings.ToLower(f)]; dbf != nil && dbf.field != "" {
			m[dbf.field] = v
		} else {
			log.Error("Invalid field in criteria", "field", f)
		}
	}
	return m
}
