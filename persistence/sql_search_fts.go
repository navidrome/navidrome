package persistence

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
)

// containsCJK returns true if the string contains any CJK (Chinese/Japanese/Korean) characters.
// CJK text doesn't use spaces between words, so FTS5's unicode61 tokenizer treats entire
// CJK phrases as single tokens, making token-based search ineffective for CJK content.
func containsCJK(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) ||
			unicode.Is(unicode.Hiragana, r) ||
			unicode.Is(unicode.Katakana, r) ||
			unicode.Is(unicode.Hangul, r) {
			return true
		}
	}
	return false
}

// fts5SpecialChars matches characters that should be stripped from user input.
// We keep only Unicode letters, numbers, whitespace, * (prefix wildcard), " (phrase quotes),
// and \x00 (internal placeholder marker). All punctuation is removed because the unicode61
// tokenizer treats it as token separators, and characters like ' can cause FTS5 parse errors
// as unbalanced string delimiters.
var fts5SpecialChars = regexp.MustCompile(`[^\p{L}\p{N}\s*"\x00]`)

// fts5PunctStrip strips everything except letters and numbers (no whitespace, wildcards, or quotes).
// Used for normalizing words at index time to create concatenated forms (e.g., "R.E.M." → "REM").
var fts5PunctStrip = regexp.MustCompile(`[^\p{L}\p{N}]`)

// fts5Operators matches FTS5 boolean operators as whole words (case-insensitive).
var fts5Operators = regexp.MustCompile(`(?i)\b(AND|OR|NOT|NEAR)\b`)

// fts5LeadingStar matches a * at the start of a token. FTS5 only supports * at the end (prefix queries).
var fts5LeadingStar = regexp.MustCompile(`(^|[\s])\*+`)

// normalizeForFTS takes multiple strings, strips non-letter/non-number characters from each word,
// and returns a space-separated string of words that changed after stripping (deduplicated).
// This is used at index time to create concatenated forms: "R.E.M." → "REM", "AC/DC" → "ACDC".
func normalizeForFTS(values ...string) string {
	seen := make(map[string]struct{})
	var result []string
	for _, v := range values {
		for _, word := range strings.Fields(v) {
			stripped := fts5PunctStrip.ReplaceAllString(word, "")
			if stripped == "" || stripped == word {
				continue
			}
			lower := strings.ToLower(stripped)
			if _, ok := seen[lower]; ok {
				continue
			}
			seen[lower] = struct{}{}
			result = append(result, stripped)
		}
	}
	return strings.Join(result, " ")
}

// isSingleUnicodeLetter returns true if token is exactly one Unicode letter.
func isSingleUnicodeLetter(token string) bool {
	r, size := utf8.DecodeRuneInString(token)
	return size == len(token) && size > 0 && unicode.IsLetter(r)
}

// namePunctuation is the set of characters commonly used as separators in artist/album
// names (hyphens, slashes, dots, apostrophes). Only words containing these are candidates
// for punctuated-word processing; other special characters (^, :, &) are just stripped.
const namePunctuation = `-/.''`

// processPunctuatedWords handles words with embedded name punctuation before the general
// special-character stripping. For each punctuated word it produces either:
//   - A quoted phrase for dotted abbreviations: R.E.M. → "R E M"
//   - A phrase+concat OR for other patterns:    a-ha  → ("a ha" OR aha*)
func processPunctuatedWords(input string, phrases []string) (string, []string) {
	words := strings.Fields(input)
	var result []string
	for _, w := range words {
		if strings.HasPrefix(w, "\x00") || strings.ContainsAny(w, `*"`) || !strings.ContainsAny(w, namePunctuation) {
			result = append(result, w)
			continue
		}
		concat := fts5PunctStrip.ReplaceAllString(w, "")
		if concat == "" || concat == w {
			result = append(result, w)
			continue
		}
		subTokens := strings.Fields(fts5SpecialChars.ReplaceAllString(w, " "))
		if len(subTokens) < 2 {
			// Single sub-token after splitting (e.g., N' → N): just use the stripped form
			result = append(result, concat)
			continue
		}
		// Dotted abbreviations (R.E.M., U.K.) — all single letters separated by dots only
		if isDottedAbbreviation(w, subTokens) {
			phrases = append(phrases, fmt.Sprintf(`"%s"`, strings.Join(subTokens, " ")))
		} else {
			// Punctuated names (a-ha, AC/DC, Jay-Z) — phrase for adjacency + concat for search_normalized
			phrases = append(phrases, fmt.Sprintf(`("%s" OR %s*)`, strings.Join(subTokens, " "), concat))
		}
		result = append(result, fmt.Sprintf("\x00PHRASE%d\x00", len(phrases)-1))
	}
	return strings.Join(result, " "), phrases
}

// isDottedAbbreviation returns true if w uses only dots as punctuation and all sub-tokens
// are single letters (e.g., "R.E.M.", "U.K." but not "a-ha" or "AC/DC").
func isDottedAbbreviation(w string, subTokens []string) bool {
	for _, r := range w {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '.' {
			return false
		}
	}
	for _, st := range subTokens {
		if !isSingleUnicodeLetter(st) {
			return false
		}
	}
	return true
}

// buildFTS5Query preprocesses user input into a safe FTS5 MATCH expression.
// It preserves quoted phrases and * prefix wildcards, neutralizes FTS5 operators
// (by lowercasing them, since FTS5 operators are case-sensitive) and strips
// special characters to prevent query injection.
func buildFTS5Query(userInput string) string {
	q := strings.TrimSpace(userInput)
	if q == "" || q == `""` {
		return ""
	}

	var phrases []string
	result := q
	for {
		start := strings.Index(result, `"`)
		if start == -1 {
			break
		}
		end := strings.Index(result[start+1:], `"`)
		if end == -1 {
			// Unmatched quote — remove it
			result = result[:start] + result[start+1:]
			break
		}
		end += start + 1
		phrase := result[start : end+1] // includes quotes
		phrases = append(phrases, phrase)
		result = result[:start] + fmt.Sprintf("\x00PHRASE%d\x00", len(phrases)-1) + result[end+1:]
	}

	// Neutralize FTS5 operators by lowercasing them (FTS5 operators are case-sensitive:
	// AND, OR, NOT, NEAR are operators, but and, or, not, near are plain tokens)
	result = fts5Operators.ReplaceAllStringFunc(result, strings.ToLower)

	// Handle words with embedded punctuation (a-ha, AC/DC, R.E.M.) before stripping
	result, phrases = processPunctuatedWords(result, phrases)

	result = fts5SpecialChars.ReplaceAllString(result, " ")
	result = fts5LeadingStar.ReplaceAllString(result, "$1")
	tokens := strings.Fields(result)

	// Append * to plain tokens for prefix matching (e.g., "love" → "love*").
	// Skip tokens that are already wildcarded or are quoted phrase placeholders.
	for i, t := range tokens {
		if strings.HasPrefix(t, "\x00") || strings.HasSuffix(t, "*") {
			continue
		}
		tokens[i] = t + "*"
	}

	result = strings.Join(tokens, " ")

	for i, phrase := range phrases {
		placeholder := fmt.Sprintf("\x00PHRASE%d\x00", i)
		result = strings.ReplaceAll(result, placeholder, phrase)
	}

	return result
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

// ftsSearchColumns defines which FTS5 columns are included in general search.
// Columns not listed here are indexed but not searched by default,
// enabling future additions (comments, lyrics, bios) without affecting general search.
var ftsSearchColumns = map[string]string{
	"media_file": "{title album artist album_artist sort_title sort_album_name sort_artist_name sort_album_artist_name disc_subtitle search_participants search_normalized}",
	"album":      "{name sort_album_name album_artist search_participants discs catalog_num album_version search_normalized}",
	"artist":     "{name sort_artist_name search_normalized}",
}

// ftsBM25Weights defines BM25 column weights for relevance ranking.
// Higher weights make matches in that column score higher. The order must match
// the column order in the FTS5 table definition (not ftsSearchColumns).
// Columns like title/name get the highest weight so exact title matches rank above
// matches in secondary fields like artist name or sort columns.
var ftsBM25Weights = map[string]string{
	// media_file_fts columns: title, album, artist, album_artist, sort_title, sort_album_name, sort_artist_name, sort_album_artist_name, disc_subtitle, search_participants, search_normalized
	"media_file": "10.0, 5.0, 3.0, 3.0, 1.0, 1.0, 1.0, 1.0, 1.0, 2.0, 1.0",
	// album_fts columns: name, sort_album_name, album_artist, search_participants, discs, catalog_num, album_version, search_normalized
	"album": "10.0, 1.0, 3.0, 2.0, 1.0, 1.0, 1.0, 1.0",
	// artist_fts columns: name, sort_artist_name, search_normalized
	"artist": "10.0, 1.0, 1.0",
}

// ftsFilter holds the information needed for a two-phase FTS5 search query.
// It implements Sqlizer so it can be used as a WHERE clause fallback, but doSearch
// detects this type and uses a more efficient two-phase approach instead.
type ftsFilter struct {
	tableName string
	ftsTable  string
	matchExpr string
	rankExpr  string // bm25() expression with column weights for ORDER BY
}

// ToSql implements Sqlizer. This is only used as a fallback when doSearch doesn't
// handle ftsFilter specially (shouldn't happen in practice).
func (f *ftsFilter) ToSql() (string, []interface{}, error) {
	sql := f.tableName + ".rowid IN (SELECT rowid FROM " + f.ftsTable + " WHERE " + f.ftsTable + " MATCH ?)"
	return sql, []interface{}{f.matchExpr}, nil
}

// ftsSearchExpr generates an FTS5 MATCH-based search filter.
// If the query produces no FTS tokens (e.g., punctuation-only like "!!!!!!!"),
// it falls back to LIKE-based search.
func ftsSearchExpr(tableName string, s string) Sqlizer {
	q := buildFTS5Query(s)
	if q == "" {
		s = strings.TrimSpace(strings.ReplaceAll(s, `"`, ""))
		if s != "" {
			log.Trace("Search using LIKE fallback for non-tokenizable query", "table", tableName, "query", s)
			return likeSearchExpr(tableName, s)
		}
		return nil
	}
	ftsTable := tableName + "_fts"
	matchExpr := q
	if cols, ok := ftsSearchColumns[tableName]; ok {
		matchExpr = cols + " : (" + q + ")"
	}

	// Build bm25() ranking expression with column weights.
	// Falls back to the built-in rank column if no weights are defined.
	rankExpr := ftsTable + ".rank"
	if weights, ok := ftsBM25Weights[tableName]; ok {
		rankExpr = "bm25(" + ftsTable + ", " + weights + ")"
	}

	filter := &ftsFilter{
		tableName: tableName,
		ftsTable:  ftsTable,
		matchExpr: matchExpr,
		rankExpr:  rankExpr,
	}
	log.Trace("Search using FTS5 backend", "table", tableName, "query", q, "filter", filter)
	return filter
}
