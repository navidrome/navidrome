// Package jsoncommentstrip provides an io.Reader that strips JavaScript-style
// comments (// line and /* block */) from JSON input while preserving
// comment-like sequences inside JSON string values.
package jsoncommentstrip

import (
	"bufio"
	"io"
)

type state int

const (
	stateNormal state = iota
	stateInString
	stateInStringEscape
	stateMaybeComment  // saw '/'
	stateLineComment   // inside // ...
	stateBlockComment  // inside /* ... */
	stateMaybeBlockEnd // saw '*' inside block comment
)

type reader struct {
	r     *bufio.Reader
	state state
}

// NewReader returns an io.Reader that strips JSON comments from the
// underlying reader. It removes single-line comments (// to end of line)
// and block comments (/* ... */), while preserving comment-like sequences
// that appear inside JSON string values.
func NewReader(r io.Reader) io.Reader {
	return &reader{
		r:     bufio.NewReader(r),
		state: stateNormal,
	}
}

func (cr *reader) Read(p []byte) (int, error) {
	n := 0
	for n < len(p) {
		b, err := cr.r.ReadByte()
		if err != nil {
			if cr.state == stateMaybeComment {
				// Emit the pending '/' before returning EOF
				p[n] = '/'
				n++
				cr.state = stateNormal
			}
			return n, err
		}

		switch cr.state {
		case stateNormal:
			switch b {
			case '"':
				p[n] = b
				n++
				cr.state = stateInString
			case '/':
				cr.state = stateMaybeComment
			default:
				p[n] = b
				n++
			}

		case stateInString:
			p[n] = b
			n++
			switch b {
			case '\\':
				cr.state = stateInStringEscape
			case '"':
				cr.state = stateNormal
			}

		case stateInStringEscape:
			p[n] = b
			n++
			cr.state = stateInString

		case stateMaybeComment:
			switch b {
			case '/':
				cr.state = stateLineComment
			case '*':
				cr.state = stateBlockComment
			default:
				// The '/' was not a comment start; emit it and the current byte
				p[n] = '/'
				n++
				if n < len(p) {
					p[n] = b
					n++
				} else {
					// We need to "unread" the current byte since buffer is full
					_ = cr.r.UnreadByte()
				}
				cr.state = stateNormal
			}

		case stateLineComment:
			if b == '\n' || b == '\r' {
				p[n] = b
				n++
				cr.state = stateNormal
			}
			// Otherwise, consume and discard

		case stateBlockComment:
			if b == '*' {
				cr.state = stateMaybeBlockEnd
			}
			// Otherwise, consume and discard

		case stateMaybeBlockEnd:
			if b == '/' {
				cr.state = stateNormal
			} else if b == '*' {
				// Stay in stateMaybeBlockEnd (consecutive *'s)
				cr.state = stateMaybeBlockEnd
			} else {
				cr.state = stateBlockComment
			}
		}
	}
	return n, nil
}
