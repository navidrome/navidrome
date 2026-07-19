package metadata

import "sync/atomic"

var genreAliases atomic.Pointer[map[string]string]

// SetGenreAliases installs the current alias_name -> canonical_name mapping, consulted by
// sanitize() while cleaning genre values during a scan. Called once per scan run (not per file)
// to avoid a DB round-trip per file.
func SetGenreAliases(aliases map[string]string) {
	genreAliases.Store(&aliases)
}

func canonicalGenre(value string) string {
	m := genreAliases.Load()
	if m == nil {
		return value
	}
	if canonical, ok := (*m)[value]; ok {
		return canonical
	}
	return value
}
