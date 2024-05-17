package tag

import (
	"io/fs"
	"sync"

	"github.com/navidrome/navidrome/model"
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

type extractorConstructor func(fs.FS, string) Extractor

var (
	extractors = map[string]extractorConstructor{}
	lock       sync.RWMutex
)

func RegisterExtractor(id string, f extractorConstructor) {
	lock.Lock()
	defer lock.Unlock()
	extractors[id] = f
}

func Extract(lib model.Library, files ...string) (map[string]Tags, error) {
	panic("not implemented")
}
