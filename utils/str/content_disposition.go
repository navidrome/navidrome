package str

import (
	"cmp"
	"fmt"
	"mime"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/deluan/sanitize"
)

const (
	maxFilenameBytes  = 255
	maxExtensionBytes = 16
	fallbackFilename  = "download"
)

// ContentDispositionAttachment builds an RFC 6266 attachment header value for a user-controlled filename.
// Non-ASCII names also get a filename* parameter, which clients prefer over the ASCII fallback.
func ContentDispositionAttachment(filename string) string {
	stem, ext := splitFilename(filename)
	header := fmt.Sprintf("attachment; filename=%q", joinFilename(toASCII(stem), toASCII(ext)))
	name := joinFilename(stem, ext)
	if isASCII(name) {
		return header
	}
	// FormatMediaType emits a percent-encoded filename* for non-ASCII values
	extended := mime.FormatMediaType("attachment", map[string]string{"filename": name})
	return header + strings.TrimPrefix(extended, "attachment")
}

func splitFilename(filename string) (stem, ext string) {
	name := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Bidi_Control, r) {
			return -1
		}
		return r
	}, SanitizeFilename(strings.ToValidUTF8(filename, "_")))
	ext = path.Ext(name)
	if len(ext) > maxExtensionBytes {
		ext = ""
	}
	stem = strings.TrimSuffix(name, ext)
	if limit := maxFilenameBytes - len(ext); len(stem) > limit {
		for limit > 0 && !utf8.RuneStart(stem[limit]) {
			limit--
		}
		stem = stem[:limit]
	}
	return stem, ext
}

func joinFilename(stem, ext string) string {
	stem = cmp.Or(strings.TrimSpace(stem), fallbackFilename)
	return cmp.Or(strings.TrimRight(stem+ext, " ."), fallbackFilename)
}

func toASCII(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == ',':
			return '_'
		case r >= ' ' && r <= '~':
			return r
		}
		return -1
	}, sanitize.Accents(SanitizeFilename(Clear(s))))
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}
