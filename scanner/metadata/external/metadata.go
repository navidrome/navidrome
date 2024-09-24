package external

import (
	"path/filepath"
)

type Tags struct {
	BasePath   string
	Pattern    string
	SetTags    map[string]string
	RemoveTags map[string]struct{}
}

func (t Tags) matchesFile(filePath string) bool {
	if t.Pattern == "" {
		return true
	}
	ok, _ := filepath.Match(t.Pattern, filePath)
	return ok
}
