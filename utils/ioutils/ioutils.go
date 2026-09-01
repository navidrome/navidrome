package ioutils

import (
	"io"
	"os"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// UTF8Reader wraps an io.Reader so downstream code always sees UTF-8, whatever
// encoding a user-provided text file (LRC lyrics, M3U playlists) happened to be
// saved in. It honors a Byte Order Mark when present: a UTF-8 BOM is stripped,
// and UTF-16 (LE/BE) is transcoded to UTF-8. When there is no BOM the bytes are
// read as UTF-8, and any byte that is not valid UTF-8 is decoded as Windows-1252
// (a superset of Latin-1). That recovers accented characters from the legacy
// single-byte encodings Windows editors still emit, instead of replacing them
// with U+FFFD and losing the match. Valid UTF-8 always passes through untouched.
//
// Reference: https://en.wikipedia.org/wiki/Byte_order_mark
func UTF8Reader(r io.Reader) io.Reader {
	return transform.NewReader(r, unicode.BOMOverride(utf8OrWindows1252{}))
}

// utf8OrWindows1252 passes valid UTF-8 through unchanged and decodes any invalid
// byte as Windows-1252. UTF-8 is preferred, so well-formed input is never
// altered; the fallback only rescues bytes that could not be valid UTF-8, which
// is the hallmark of a legacy Latin-1/Windows-1252 file.
type utf8OrWindows1252 struct{}

func (utf8OrWindows1252) Reset() {}

func (utf8OrWindows1252) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	for nSrc < len(src) {
		b := src[nSrc]

		// ASCII fast path: identical in UTF-8 and Windows-1252.
		if b < utf8.RuneSelf {
			if nDst >= len(dst) {
				return nDst, nSrc, transform.ErrShortDst
			}
			dst[nDst] = b
			nDst++
			nSrc++
			continue
		}

		// A byte >= 0x80 begins a multi-byte UTF-8 sequence. If the remaining
		// input might still hold an incomplete sequence, ask for more before
		// judging it invalid, so a rune split across reads is not misdecoded.
		if !atEOF && !utf8.FullRune(src[nSrc:]) {
			return nDst, nSrc, transform.ErrShortSrc
		}

		if r, size := utf8.DecodeRune(src[nSrc:]); r != utf8.RuneError || size > 1 {
			// Valid UTF-8 rune (including a genuine U+FFFD): copy it verbatim.
			if nDst+size > len(dst) {
				return nDst, nSrc, transform.ErrShortDst
			}
			copy(dst[nDst:], src[nSrc:nSrc+size])
			nDst += size
			nSrc += size
			continue
		}

		// Not valid UTF-8: decode this single byte as Windows-1252.
		r := charmap.Windows1252.DecodeByte(b)
		if nDst+utf8.RuneLen(r) > len(dst) {
			return nDst, nSrc, transform.ErrShortDst
		}
		nDst += utf8.EncodeRune(dst[nDst:], r)
		nSrc++
	}
	return nDst, nSrc, nil
}

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
