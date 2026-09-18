package str

import (
	"fmt"
	"strings"
)

// fallbackFilename is used when sanitizing leaves nothing usable (e.g. a name made
// entirely of non-ASCII runes). Clients that understand filename* still get the real
// name; this is only the ASCII fallback.
const fallbackFilename = "download"

// ContentDispositionAttachment builds a Content-Disposition header value (RFC 6266)
// for an attachment download named filename.
//
// Filenames here are derived from user-controlled data — playlist names, album and
// artist names, and titles read from media file tags — so they must never be
// interpolated into the header as-is. A name containing a double quote closes the
// quoted-string early and the remainder is parsed as additional parameters, so a
// playlist named
//
//	party"; filename="evil.html
//
// would produce `attachment; filename="party"; filename="evil.html.m3u"`, letting
// whoever chose the name control what the browser saves the download as. Go's
// net/http rewrites CR and LF in header values to spaces, so response splitting is
// not reachable this way, but parameter injection is.
//
// The `filename` parameter is reduced to a quoted, ASCII-only token and the original
// UTF-8 name is preserved in an RFC 5987 `filename*` parameter, which RFC 6266 §4.3
// requires clients to prefer when both are present. Without `filename*`, sanitizing
// to ASCII would mangle every non-Latin name.
func ContentDispositionAttachment(filename string) string {
	return fmt.Sprintf("attachment; filename=%q; filename*=UTF-8''%s",
		asciiFilename(filename), encodeRFC5987(filename))
}

// asciiFilename reduces filename to a token that is safe inside a quoted-string: the
// characters SanitizeFilename already strips (including the double quote), plus
// commas, control characters and everything outside printable ASCII. Leading and
// trailing spaces and dots are trimmed as well, since they are not valid in Windows
// filenames.
func asciiFilename(filename string) string {
	var sb strings.Builder
	sb.Grow(len(filename))
	for _, r := range SanitizeFilename(filename) {
		switch {
		case r == ',':
			// Some clients split the header value on commas
			sb.WriteByte('_')
		case r >= ' ' && r <= '~':
			sb.WriteRune(r)
		}
	}
	name := strings.Trim(sb.String(), " .")
	if name == "" {
		return fallbackFilename
	}
	return name
}

// encodeRFC5987 percent-encodes filename for use as an ext-value, escaping every byte
// outside the attr-char set defined in RFC 5987 §3.2.1.
func encodeRFC5987(filename string) string {
	const upperhex = "0123456789ABCDEF"
	var sb strings.Builder
	sb.Grow(len(filename))
	for i := 0; i < len(filename); i++ {
		if c := filename[i]; isAttrChar(c) {
			sb.WriteByte(c)
		} else {
			sb.WriteByte('%')
			sb.WriteByte(upperhex[c>>4])
			sb.WriteByte(upperhex[c&0x0F])
		}
	}
	return sb.String()
}

// isAttrChar reports whether c is an attr-char: RFC 5987 §3.2.1 defines it as
// ALPHA / DIGIT / "!" / "#" / "$" / "&" / "+" / "-" / "." / "^" / "_" / "`" / "|" / "~"
func isAttrChar(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	}
	return strings.IndexByte("!#$&+-.^_`|~", c) >= 0
}
