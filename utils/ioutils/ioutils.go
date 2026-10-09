package ioutils

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// UTF8Reader wraps an io.Reader so downstream code always sees UTF-8, whatever
// encoding a user-provided text file (LRC lyrics, M3U playlists) happened to be
// saved in. A Byte Order Mark is authoritative: a UTF-8 BOM is stripped and
// UTF-16 (LE/BE) is transcoded to UTF-8, both handled as a stream.
//
// Without a BOM the encoding has to be judged from the whole content. The reader
// keeps the bytes as they are when they form valid UTF-8, and otherwise decodes
// the entire input as Windows-1252 (a superset of Latin-1). Choosing one encoding
// for the whole file, rather than per character, matters: a genuinely
// Windows-1252 file can contain a byte pair that happens to be valid UTF-8 (for
// example "Â£" is C2 A3), and a per-character choice would decode that pair as a
// different string than the rest of the file, so the affected path would silently
// fail to match. This recovers accented characters from the legacy single-byte
// encodings Windows editors still emit instead of replacing them with U+FFFD.
//
// Reference: https://en.wikipedia.org/wiki/Byte_order_mark
func UTF8Reader(r io.Reader) io.Reader {
	br := bufio.NewReader(r)

	// A BOM is an explicit declaration, so honor it and keep the stream lazy.
	if prefix, _ := br.Peek(3); hasBOM(prefix) {
		return transform.NewReader(br, unicode.BOMOverride(unicode.UTF8.NewDecoder()))
	}

	// No BOM: read it all so the encoding can be decided from the whole content.
	data, err := io.ReadAll(br)
	out := data
	if !utf8.Valid(data) {
		// Windows-1252 maps every byte, so this decode never fails.
		out, _ = charmap.Windows1252.NewDecoder().Bytes(data)
	}
	if err != nil {
		return io.MultiReader(bytes.NewReader(out), errorReader{err: err})
	}
	return bytes.NewReader(out)
}

// hasBOM reports whether b starts with a UTF-8 or UTF-16 (LE/BE) Byte Order Mark.
func hasBOM(b []byte) bool {
	switch {
	case len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF: // UTF-8
		return true
	case len(b) >= 2 && b[0] == 0xFF && b[1] == 0xFE: // UTF-16 LE
		return true
	case len(b) >= 2 && b[0] == 0xFE && b[1] == 0xFF: // UTF-16 BE
		return true
	default:
		return false
	}
}

// errorReader replays a read error that surfaced while buffering the input, so
// callers still see it lazily on their next Read.
type errorReader struct{ err error }

func (e errorReader) Read([]byte) (int, error) { return 0, e.err }

// UTF8ReadFile reads the named file and returns its contents as UTF-8 bytes.
// It's like os.ReadFile but runs the data through UTF8Reader, so BOMs are
// stripped, UTF-16 is transcoded, and legacy Windows-1252/Latin-1 bytes are
// recovered rather than replaced with U+FFFD.
func UTF8ReadFile(filename string) ([]byte, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := UTF8Reader(file)
	return io.ReadAll(reader)
}
