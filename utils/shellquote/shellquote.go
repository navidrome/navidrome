package shellquote

import (
	"errors"
	"strings"
)

var (
	ErrUnterminatedSingleQuote = errors.New("unterminated single-quoted string")
	ErrUnterminatedDoubleQuote = errors.New("unterminated double-quoted string")
	ErrUnterminatedEscape      = errors.New("unterminated backslash-escape")
)

type state int

const (
	stateUnquoted state = iota
	stateSingleQuoted
	stateDoubleQuoted
)

// Split splits a string into words following POSIX-like shell quoting rules.
// It handles single quotes, double quotes, and backslash escapes.
func Split(input string) ([]string, error) {
	var words []string
	var word strings.Builder
	inWord := false
	parseState := stateUnquoted

	i := 0
	for i < len(input) {
		ch := input[i]

		switch parseState {
		case stateUnquoted:
			switch {
			case ch == '\\':
				if i+1 >= len(input) {
					return nil, ErrUnterminatedEscape
				}
				if input[i+1] == '\n' {
					// Line continuation: skip both backslash and newline
					i += 2
					continue
				}
				i++
				word.WriteByte(input[i])
				inWord = true
			case ch == '\'':
				parseState = stateSingleQuoted
				inWord = true
			case ch == '"':
				parseState = stateDoubleQuoted
				inWord = true
			case ch == ' ' || ch == '\t' || ch == '\n':
				if inWord {
					words = append(words, word.String())
					word.Reset()
					inWord = false
				}
			default:
				word.WriteByte(ch)
				inWord = true
			}

		case stateSingleQuoted:
			if ch == '\'' {
				parseState = stateUnquoted
			} else {
				word.WriteByte(ch)
			}

		case stateDoubleQuoted:
			switch {
			case ch == '"':
				parseState = stateUnquoted
			case ch == '\\':
				if i+1 >= len(input) {
					return nil, ErrUnterminatedEscape
				}
				next := input[i+1]
				// In double quotes, backslash only escapes: $ ` " \n \
				if next == '$' || next == '`' || next == '"' || next == '\n' || next == '\\' {
					if next == '\n' {
						// Line continuation: skip both backslash and newline
						i += 2
						continue
					}
					i++
					word.WriteByte(next)
				} else {
					// Backslash is literal for other characters
					word.WriteByte(ch)
				}
			default:
				word.WriteByte(ch)
			}
		}

		i++
	}

	switch parseState {
	case stateSingleQuoted:
		return nil, ErrUnterminatedSingleQuote
	case stateDoubleQuoted:
		return nil, ErrUnterminatedDoubleQuote
	}

	if inWord {
		words = append(words, word.String())
	}

	return words, nil
}
