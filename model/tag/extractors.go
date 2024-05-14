package tag

import (
	"sync"
)

type Extractor interface {
	Parse(files ...string) (map[string]Properties, error)
	Version() string
}

type Properties struct {
	Tags            map[string][]string
	AudioProperties AudioProperties
	HasPicture      bool
}

var (
	extractors = map[string]Extractor{}
	lock       sync.RWMutex
)

func RegisterExtractor(id string, parser Extractor) {
	lock.Lock()
	defer lock.Unlock()
	extractors[id] = parser
}

func getExtractor(id string) Extractor {
	lock.RLock()
	defer lock.RUnlock()
	return extractors[id]
}

func Extract(files ...string) (map[string]Tags, error) {
	panic("not implemented")
}
