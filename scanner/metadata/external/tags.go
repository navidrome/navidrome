package external

import (
	"github.com/bmatcuk/doublestar/v4"
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
	ok, err := doublestar.Match(t.Pattern, filePath)
	return err == nil && ok
}
