package persistence

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
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

// ftsColumn pairs an FTS5 column name with its BM25 relevance weight.
type ftsColumn struct {
	Name   string
	Weight float64
}

// ftsColumnDefs defines FTS5 columns and their BM25 relevance weights.
// The order MUST match the column order in the FTS5 table definition (see migrations).
// All columns are both searched and ranked. When adding indexed-but-not-searched
// columns in the future, use Weight: 0 to exclude from the search column filter.
var ftsColumnDefs = map[string][]ftsColumn{
	"media_file": {
		{"title", 10.0},
		{"album", 5.0},
		{"artist", 3.0},
		{"album_artist", 3.0},
		{"sort_title", 1.0},
		{"sort_album_name", 1.0},
		{"sort_artist_name", 1.0},
		{"sort_album_artist_name", 1.0},
		{"disc_subtitle", 1.0},
		{"search_participants", 2.0},
		{"search_normalized", 1.0},
	},
	"album": {
		{"name", 10.0},
		{"sort_album_name", 1.0},
		{"album_artist", 3.0},
		{"search_participants", 2.0},
		{"discs", 1.0},
		{"catalog_num", 1.0},
		{"album_version", 1.0},
		{"search_normalized", 1.0},
	},
	"artist": {
		{"name", 10.0},
		{"sort_artist_name", 1.0},
		{"search_normalized", 1.0},
	},
}

// ftsColumnFilters and ftsBM25Weights are precomputed from ftsColumnDefs at init time
// to avoid per-query allocations.
var (
	ftsColumnFilters = map[string]string{}
	ftsBM25Weights   = map[string]string{}
)

func init() {
	for table, cols := range ftsColumnDefs {
		var names []string
		weights := make([]string, len(cols))
		for i, c := range cols {
			if c.Weight > 0 {
				names = append(names, c.Name)
			}
			weights[i] = fmt.Sprintf("%.1f", c.Weight)
		}
		ftsColumnFilters[table] = "{" + strings.Join(names, " ") + "}"
		ftsBM25Weights[table] = strings.Join(weights, ", ")
	}
}

// ftsSearch implements searchStrategy using FTS5 full-text search with BM25 ranking.
type ftsSearch struct {
	tableName string
	ftsTable  string
	matchExpr string
	rankExpr  string
}

// ToSql returns a single-query fallback for the REST filter path (no two-phase split).
func (s *ftsSearch) ToSql() (string, []interface{}, error) {
	sql := s.tableName + ".rowid IN (SELECT rowid FROM " + s.ftsTable + " WHERE " + s.ftsTable + " MATCH ?)"
	return sql, []interface{}{s.matchExpr}, nil
}

// execute runs a two-phase FTS5 search:
//   - Phase 1: lightweight rowid query (main table + FTS + library filter) for ranking and pagination.
//   - Phase 2: full SELECT with all JOINs, scoped to Phase 1's rowid set.
//
// Complex ORDER BY (function calls, aggregations) are dropped from Phase 1.
func (s *ftsSearch) execute(r sqlRepository, sq SelectBuilder, dest any, cfg searchConfig, options model.QueryOptions) error {
	qualifiedOrderBys := []string{s.rankExpr}
	for _, ob := range cfg.OrderBy {
		if qualified := qualifyOrderBy(s.tableName, ob); qualified != "" {
			qualifiedOrderBys = append(qualifiedOrderBys, qualified)
		}
	}

	// Phase 1: fresh query — must set LIMIT/OFFSET from options explicitly.
	// Mirror applyOptions behavior: Max=0 means no limit, not LIMIT 0.
	rowidQuery := Select(s.tableName+".rowid").
		From(s.tableName).
		Join(s.ftsTable+" ON "+s.ftsTable+".rowid = "+s.tableName+".rowid AND "+s.ftsTable+" MATCH ?", s.matchExpr).
		Where(Eq{s.tableName + ".missing": false}).
		OrderBy(qualifiedOrderBys...)
	if options.Max > 0 {
		rowidQuery = rowidQuery.Limit(uint64(options.Max))
	}
	if options.Offset > 0 {
		rowidQuery = rowidQuery.Offset(uint64(options.Offset))
	}

	// Library filter + musicFolderId must be applied here, before pagination.
	if cfg.LibraryFilter != nil {
		rowidQuery = cfg.LibraryFilter(rowidQuery)
	} else {
		rowidQuery = r.applyLibraryFilter(rowidQuery)
	}
	if options.Filters != nil {
		rowidQuery = rowidQuery.Where(options.Filters)
	}

	rowidSQL, rowidArgs, err := rowidQuery.ToSql()
	if err != nil {
		return fmt.Errorf("building FTS rowid query: %w", err)
	}

	// Phase 2: strip LIMIT/OFFSET from sq (Phase 1 handled pagination),
	// join on the ranked rowid set to hydrate with full columns.
	sq = sq.RemoveLimit().RemoveOffset()
	rankedSubquery := fmt.Sprintf(
		"(SELECT rowid as _rid, row_number() OVER () AS _rn FROM (%s)) AS _ranked",
		rowidSQL,
	)
	sq = sq.Join(rankedSubquery+" ON "+s.tableName+".rowid = _ranked._rid", rowidArgs...)
	sq = sq.OrderBy("_ranked._rn")
	return r.queryAll(sq, dest)
}

// qualifyOrderBy prepends tableName to a simple column name. Returns empty string for
// complex expressions (function calls, aggregations) that can't be used in Phase 1.
func qualifyOrderBy(tableName, orderBy string) string {
	orderBy = strings.TrimSpace(orderBy)
	if orderBy == "" || strings.ContainsAny(orderBy, "(,") {
		return ""
	}
	parts := strings.Fields(orderBy)
	if !strings.Contains(parts[0], ".") {
		parts[0] = tableName + "." + parts[0]
	}
	return strings.Join(parts, " ")
}

// ftsQueryDegraded returns true when the FTS query lost significant discriminating
// content compared to the original input. This happens when special characters that
// are part of the entity name (e.g., "1+", "C++", "!!!", "C#") get stripped by FTS
// tokenization, leaving only very short/broad tokens. Also detects quoted phrases
// that would be degraded by FTS5's unicode61 tokenizer (e.g., "1+" → token "1").
func ftsQueryDegraded(original, ftsQuery string) bool {
	original = strings.TrimSpace(original)
	if original == "" || ftsQuery == "" {
		return false
	}
	// Strip quotes from original for comparison — we want the raw content
	stripped := strings.ReplaceAll(original, `"`, "")
	// Extract the alphanumeric content from the original query
	alphaNum := fts5PunctStrip.ReplaceAllString(stripped, "")
	// If the original is entirely alphanumeric, nothing was stripped — not degraded
	if len(alphaNum) == len(stripped) {
		return false
	}
	// Check if all effective FTS tokens are very short (≤2 chars).
	// Short tokens with prefix matching are too broad when special chars were stripped.
	// For quoted phrases, extract the content and check the tokens inside.
	tokens := strings.Fields(ftsQuery)
	for _, t := range tokens {
		t = strings.TrimSuffix(t, "*")
		// Skip internal phrase placeholders
		if strings.HasPrefix(t, "\x00") {
			return false
		}
		// For OR groups from processPunctuatedWords (e.g., ("a ha" OR aha*)),
		// the punctuated word was already handled meaningfully — not degraded.
		if strings.HasPrefix(t, "(") {
			return false
		}
		// For quoted phrases, check the tokens inside as FTS5 will tokenize them
		if strings.HasPrefix(t, `"`) {
			// Extract content between quotes
			inner := strings.Trim(t, `"`)
			innerAlpha := fts5PunctStrip.ReplaceAllString(inner, " ")
			for _, it := range strings.Fields(innerAlpha) {
				if len(it) > 2 {
					return false
				}
			}
			continue
		}
		if len(t) > 2 {
			return false
		}
	}
	return true
}

// newFTSSearch creates an FTS5 search strategy. Falls back to LIKE search if the
// query produces no FTS tokens (e.g., punctuation-only like "!!!!!!!") or if FTS
// tokenization stripped significant content from the query (e.g., "1+" → "1*").
// Returns nil when the query produces no searchable tokens at all.
func newFTSSearch(tableName, query string) searchStrategy {
	q := buildFTS5Query(query)
	if q == "" || ftsQueryDegraded(query, q) {
		// Fallback: try LIKE search with the raw query
		cleaned := strings.TrimSpace(strings.ReplaceAll(query, `"`, ""))
		if cleaned != "" {
			log.Trace("Search using LIKE fallback for non-tokenizable query", "table", tableName, "query", cleaned)
			return newLikeSearch(tableName, cleaned)
		}
		return nil
	}
	ftsTable := tableName + "_fts"
	matchExpr := q
	if cols, ok := ftsColumnFilters[tableName]; ok {
		matchExpr = cols + " : (" + q + ")"
	}

	rankExpr := ftsTable + ".rank"
	if weights, ok := ftsBM25Weights[tableName]; ok {
		rankExpr = "bm25(" + ftsTable + ", " + weights + ")"
	}

	s := &ftsSearch{
		tableName: tableName,
		ftsTable:  ftsTable,
		matchExpr: matchExpr,
		rankExpr:  rankExpr,
	}
	log.Trace("Search using FTS5 backend", "table", tableName, "query", q, "filter", s)
	return s
}
