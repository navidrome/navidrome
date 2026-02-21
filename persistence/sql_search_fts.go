package persistence

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	. "github.com/Masterminds/squirrel"
)

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

// namePunctuation matches characters commonly found as separators in artist/album names.
// Only words containing these characters are candidates for punctuated-word handling.
var namePunctuation = regexp.MustCompile(`[-/.'']`)

// isDottedAbbreviation returns true if w is a dot-separated abbreviation where all
// sub-tokens are single letters (e.g., "R.E.M.", "U.K."). These are handled by
// collapseSingleLetterRuns instead of extractPunctuatedWords.
func isDottedAbbreviation(w string, subTokens []string) bool {
	// Check that punctuation is only dots
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

// extractPunctuatedWords scans whitespace-delimited words for embedded name punctuation
// (hyphens, slashes, dots, apostrophes — e.g., "a-ha", "AC/DC", "Jay-Z") and replaces
// each with an OR expression: ("a ha" OR aha*).
// The phrase matches adjacent tokens in name columns; the concatenated form matches
// the search_normalized column.
// Words where all sub-tokens are single letters (e.g., "R.E.M.") are skipped
// and left for collapseSingleLetterRuns to handle.
func extractPunctuatedWords(input string, phrases []string) (string, []string) {
	words := strings.Fields(input)
	var result []string
	for _, w := range words {
		// Skip phrase placeholders
		if strings.HasPrefix(w, "\x00") {
			result = append(result, w)
			continue
		}
		// Skip tokens that contain wildcards or quotes (already processed)
		if strings.ContainsAny(w, `*"`) {
			result = append(result, w)
			continue
		}
		// Only process words that contain common name punctuation
		if !namePunctuation.MatchString(w) {
			result = append(result, w)
			continue
		}
		// Strip all punctuation to get concatenated form
		concat := fts5PunctStrip.ReplaceAllString(w, "")
		if concat == "" || concat == w {
			result = append(result, w)
			continue
		}
		// Get the space-separated form (what fts5SpecialChars would produce)
		spaced := fts5SpecialChars.ReplaceAllString(w, " ")
		subTokens := strings.Fields(spaced)
		if len(subTokens) < 2 {
			// Single sub-token (e.g., N' → N): replace inline, normal processing continues
			result = append(result, concat)
			continue
		}
		// If all sub-tokens are single letters and the word only contains dots as
		// punctuation (e.g., R.E.M.), skip — collapseSingleLetterRuns handles these.
		// For other separators (a-h, A/B), handle here so the concat form provides
		// prefix matching (e.g., ah* matches "aha" in search_normalized).
		if isDottedAbbreviation(w, subTokens) {
			result = append(result, w)
			continue
		}
		// Build OR expression: ("a ha" OR aha*)
		phraseContent := strings.Join(subTokens, " ")
		orExpr := fmt.Sprintf(`("%s" OR %s*)`, phraseContent, concat)
		phrases = append(phrases, orExpr)
		placeholder := fmt.Sprintf("\x00PHRASE%d\x00", len(phrases)-1)
		result = append(result, placeholder)
	}
	return strings.Join(result, " "), phrases
}

// collapseSingleLetterRuns scans tokens for runs of 2+ consecutive single Unicode letters
// (from abbreviations like R.E.M. where all segments are single letters) and collapses
// each run into a quoted phrase: "R E M".
// The phrase matches consecutive tokens in name columns and the concatenated form in
// the search_normalized column.
func collapseSingleLetterRuns(tokens []string, phrases []string) ([]string, []string) {
	var result []string
	i := 0
	for i < len(tokens) {
		// Skip phrase placeholders
		if strings.HasPrefix(tokens[i], "\x00") {
			result = append(result, tokens[i])
			i++
			continue
		}
		// Detect start of a single-letter run
		if isSingleUnicodeLetter(tokens[i]) {
			j := i + 1
			for j < len(tokens) && !strings.HasPrefix(tokens[j], "\x00") && isSingleUnicodeLetter(tokens[j]) {
				j++
			}
			if j-i >= 2 {
				// Collapse the run into a quoted phrase (e.g., R.E.M. → "R E M")
				letters := tokens[i:j]
				phraseContent := strings.Join(letters, " ")
				phrase := fmt.Sprintf(`"%s"`, phraseContent)
				// Store the phrase as a placeholder
				phrases = append(phrases, phrase)
				placeholder := fmt.Sprintf("\x00PHRASE%d\x00", len(phrases)-1)
				result = append(result, placeholder)
				i = j
				continue
			}
		}
		result = append(result, tokens[i])
		i++
	}
	return result, phrases
}

// buildFTS5Query preprocesses user input into a safe FTS5 MATCH expression.
// It preserves quoted phrases and * prefix wildcards, neutralizes FTS5 operators
// (by lowercasing them, since FTS5 operators are case-sensitive) and strips
// special characters to prevent query injection.
func buildFTS5Query(userInput string) string {
	q := strings.TrimSpace(userInput)
	if q == "" {
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

	// Detect words with embedded punctuation (a-ha, AC/DC, Jay-Z) before stripping,
	// and replace them with phrase+concat OR expressions.
	result, phrases = extractPunctuatedWords(result, phrases)

	result = fts5SpecialChars.ReplaceAllString(result, " ")
	result = fts5LeadingStar.ReplaceAllString(result, "$1")
	tokens := strings.Fields(result)

	// Collapse runs of single letters (from abbreviations like R.E.M.) into quoted phrases
	tokens, phrases = collapseSingleLetterRuns(tokens, phrases)

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

// ftsSearchColumns defines which FTS5 columns are included in general search.
// Columns not listed here are indexed but not searched by default,
// enabling future additions (comments, lyrics, bios) without affecting general search.
var ftsSearchColumns = map[string]string{
	"media_file": "{title album artist album_artist sort_title sort_album_name sort_artist_name sort_album_artist_name disc_subtitle search_participants search_normalized}",
	"album":      "{name sort_album_name album_artist search_participants discs catalog_num search_normalized}",
	"artist":     "{name sort_artist_name search_normalized}",
}

// ftsSearchExpr generates an FTS5 MATCH-based search filter.
func ftsSearchExpr(tableName string, s string) Sqlizer {
	q := buildFTS5Query(s)
	if q == "" {
		return nil
	}
	ftsTable := tableName + "_fts"
	matchExpr := q
	if cols, ok := ftsSearchColumns[tableName]; ok {
		matchExpr = cols + " : (" + q + ")"
	}

	return Expr(
		tableName+".rowid IN (SELECT rowid FROM "+ftsTable+" WHERE "+ftsTable+" MATCH ?)",
		matchExpr,
	)
}
