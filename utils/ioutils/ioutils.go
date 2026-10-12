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
// Without a BOM, each line is judged on its own. A line that is valid UTF-8 is
// kept as it is, and any other line is decoded as Windows-1252 (a superset of
// Latin-1). Both file formats we read are line oriented, so a line is the
// natural unit. It also keeps one stray legacy line, say a track appended by an
// old editor, from garbling every other path in an otherwise UTF-8 file. The
// trade-off is a Windows-1252 line whose high bytes happen to form valid UTF-8
// (for example "Â£" is C2 A3) and has no other accented character: it is read
// as UTF-8, because nothing on that line says otherwise.
//
// Reference: https://en.wikipedia.org/wiki/Byte_order_mark
func UTF8Reader(r io.Reader) io.Reader {
	return UTF8ReaderWithFallback(r, nil)
}

// UTF8ReaderWithFallback works like UTF8Reader, and calls onFallback once with
// the number of lines it had to decode as Windows-1252. Callers use it to log
// that a file is not UTF-8, which would otherwise go unnoticed. The callback
// runs before this function returns and is skipped when no line needed it, or
// when onFallback is nil. It is never called for input that starts with a BOM.
func UTF8ReaderWithFallback(r io.Reader, onFallback func(lines int)) io.Reader {
	br := bufio.NewReader(r)

	// A BOM is an explicit declaration, so honor it and keep the stream lazy.
	if prefix, _ := br.Peek(3); hasBOM(prefix) {
		return transform.NewReader(br, unicode.BOMOverride(unicode.UTF8.NewDecoder()))
	}

	// No BOM: read it all so each line can be checked.
	data, err := io.ReadAll(br)
	out, fallbackLines := decodeLines(data)
	if fallbackLines > 0 && onFallback != nil {
		onFallback(fallbackLines)
	}
	if err != nil {
		return io.MultiReader(bytes.NewReader(out), errorReader{err: err})
	}
	return bytes.NewReader(out)
}

// decodeLines keeps the lines of data that are valid UTF-8 and decodes the rest
// as Windows-1252. It returns the result and how many lines were decoded.
func decodeLines(data []byte) ([]byte, int) {
	if utf8.Valid(data) {
		return data, 0
	}

	decoder := charmap.Windows1252.NewDecoder()
	out := make([]byte, 0, len(data))
	fallbackLines := 0
	for len(data) > 0 {
		line := data
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			line = data[:i+1]
		}
		data = data[len(line):]

		if utf8.Valid(line) {
			out = append(out, line...)
			continue
		}
		// Windows-1252 maps every byte, so this decode never fails.
		decoded, _ := decoder.Bytes(line)
		out = append(out, decoded...)
		fallbackLines++
	}
	return out, fallbackLines
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
