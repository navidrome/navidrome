package persistence

import (
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/str"
)

// likeSearch implements searchStrategy using LIKE-based SQL filters.
// Used for legacy full_text searches, CJK fallback, and punctuation-only fallback.
type likeSearch struct {
	filter Sqlizer
}

func (s *likeSearch) ToSql() (string, []interface{}, error) {
	return s.filter.ToSql()
}

func (s *likeSearch) execute(r sqlRepository, sq SelectBuilder, dest any, cfg searchConfig, options model.QueryOptions) error {
	sq = sq.Where(s.filter)
	sq = sq.OrderBy(cfg.OrderBy...)
	return r.queryAll(sq, dest, options)
}

// newLegacySearch creates a LIKE search against the full_text column.
// Returns nil when the query produces no searchable tokens.
func newLegacySearch(tableName, query string) searchStrategy {
	filter := legacySearchExpr(tableName, query)
	if filter == nil {
		return nil
	}
	return &likeSearch{filter: filter}
}

// newLikeSearch creates a LIKE search against core entity columns (CJK, punctuation fallback).
// No minimum length is enforced, since single CJK characters are meaningful words.
// Returns nil when the query produces no searchable tokens.
func newLikeSearch(tableName, query string) searchStrategy {
	filter := likeSearchExpr(tableName, query)
	if filter == nil {
		return nil
	}
	return &likeSearch{filter: filter}
}

// legacySearchExpr generates LIKE-based search filters against the full_text column.
// This is the original search implementation, used when Search.Backend="legacy".
func legacySearchExpr(tableName string, s string) Sqlizer {
	q := str.SanitizeStrings(s)
	if q == "" {
		log.Trace("Search using legacy backend, query is empty", "table", tableName)
		return nil
	}
	var sep string
	if !conf.Server.Search.FullString {
		sep = " "
	}
	parts := strings.Split(q, " ")
	filters := And{}
	for _, part := range parts {
		filters = append(filters, Like{tableName + ".full_text": "%" + sep + part + "%"})
	}
	log.Trace("Search using legacy backend", "query", filters, "table", tableName)
	return filters
}

// likeSearchColumns defines the core columns to search with LIKE queries.
// These are the primary user-visible fields for each entity type.
// Used as a fallback when FTS5 cannot handle the query (e.g., CJK text, punctuation-only input).
var likeSearchColumns = map[string][]string{
	"media_file": {"title", "album", "artist", "album_artist"},
	"album":      {"name", "album_artist"},
	"artist":     {"name"},
}

// likeSearchExpr generates LIKE-based search filters against core columns.
// Each word in the query must match at least one column (AND between words),
// and each word can match any column (OR within a word).
// Used as a fallback when FTS5 cannot handle the query (e.g., CJK text, punctuation-only input).
func likeSearchExpr(tableName string, s string) Sqlizer {
	s = strings.TrimSpace(s)
	if s == "" {
		log.Trace("Search using LIKE backend, query is empty", "table", tableName)
		return nil
	}
	columns, ok := likeSearchColumns[tableName]
	if !ok {
		log.Trace("Search using LIKE backend, couldn't find columns for this table", "table", tableName)
		return nil
	}
	words := strings.Fields(s)
	wordFilters := And{}
	for _, word := range words {
		colFilters := Or{}
		for _, col := range columns {
			colFilters = append(colFilters, Like{tableName + "." + col: "%" + word + "%"})
		}
		wordFilters = append(wordFilters, colFilters)
	}
	log.Trace("Search using LIKE backend", "query", wordFilters, "table", tableName)
	return wordFilters
}
