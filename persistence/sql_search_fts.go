package persistence

import (
	"fmt"
	"regexp"
	"strings"

	. "github.com/Masterminds/squirrel"
)

// fts5SpecialChars matches FTS5 operator characters that must be stripped from user input.
// This includes: ^ (column weight), : (column filter), + (token position), - (unary NOT).
// Parentheses are also removed to prevent grouping injection.
var fts5SpecialChars = regexp.MustCompile(`[():^+\-]`)

// fts5Operators matches FTS5 boolean operators as whole words (case-insensitive).
var fts5Operators = regexp.MustCompile(`(?i)\b(AND|OR|NOT|NEAR)\b`)

// fts5LeadingStar matches a * at the start of a token. FTS5 only supports * at the end (prefix queries).
var fts5LeadingStar = regexp.MustCompile(`(^|[\s])\*+`)

// buildFTS5Query preprocesses user input into a safe FTS5 MATCH expression.
// It preserves quoted phrases and * prefix wildcards, neutralizes FTS5 operators
// (by lowercasing them, since FTS5 operators are case-sensitive) and strips
// special characters to prevent query injection.
func buildFTS5Query(userInput string) string {
	q := strings.TrimSpace(userInput)
	if q == "" {
		return ""
	}

	// Extract quoted phrases first, replace with placeholders
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

	// Strip special FTS5 characters (but keep * for prefix queries)
	result = fts5SpecialChars.ReplaceAllString(result, " ")

	// Remove leading * from tokens (FTS5 only supports trailing * for prefix queries)
	result = fts5LeadingStar.ReplaceAllString(result, "$1")

	// Collapse whitespace and split into tokens
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

	// Restore phrases
	for i, phrase := range phrases {
		placeholder := fmt.Sprintf("\x00PHRASE%d\x00", i)
		result = strings.ReplaceAll(result, placeholder, phrase)
	}

	return result
}

// ftsSearchExpr generates an FTS5 MATCH-based search filter.
// Returns a subquery: `tableName.rowid IN (SELECT rowid FROM tableName_fts WHERE tableName_fts MATCH ?)`.
func ftsSearchExpr(tableName string, s string) Sqlizer {
	q := buildFTS5Query(s)
	if q == "" {
		return nil
	}
	ftsTable := tableName + "_fts"

	return Expr(
		tableName+".rowid IN (SELECT rowid FROM "+ftsTable+" WHERE "+ftsTable+" MATCH ?)",
		q,
	)
}
