package str

import (
	"regexp"
	"strings"

	"github.com/deluan/sanitize"
)

// FTSPunctStrip matches any character that is not a letter or number. Index-time
// normalization (NormalizeForFTS) and query-time processing in persistence share it
// so both sides produce matching tokens.
var FTSPunctStrip = regexp.MustCompile(`[^\p{L}\p{N}]`)

// NormalizeForFTS takes multiple strings and returns a space-separated, deduplicated list of
// alternative searchable forms for each word: punctuation-stripped (R.E.M. → REM, AC/DC → ACDC)
// and ASCII-transliterated (Bjørk → Bjork, œuvre → oeuvre). The transliterated form is needed
// because FTS5's `unicode61 remove_diacritics 2` only handles NFKD-decomposable diacritics —
// atomic letters like ø/æ/œ/ß survive tokenization, so the query side and index side disagree
// without an explicit transliterated entry here.
func NormalizeForFTS(values ...string) string {
	seen := make(map[string]struct{})
	var result []string
	add := func(orig, variant string) {
		if variant == "" || variant == orig {
			return
		}
		lower := strings.ToLower(variant)
		if _, ok := seen[lower]; ok {
			return
		}
		seen[lower] = struct{}{}
		result = append(result, variant)
	}
	for _, v := range values {
		for word := range strings.FieldsSeq(v) {
			transliterated := sanitize.Accents(word)
			// Concatenated ASCII form: R.E.M. → REM, AC/DC → ACDC, St-Étienne → StEtienne.
			add(word, FTSPunctStrip.ReplaceAllString(transliterated, ""))
			// Accent-only transliteration for words without name-punctuation (Bjørk → Bjork).
			add(word, transliterated)
		}
	}
	return strings.Join(result, " ")
}
