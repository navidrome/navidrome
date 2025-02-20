package local

import (
	"io/fs"
	"sync"

	"github.com/navidrome/navidrome/model/metadata"
)

// Extractor is an interface that defines the methods that a tag/metadata extractor must implement
type Extractor interface {
	Parse(files ...string) (map[string]metadata.Info, error)
	Version() string
}

type extractorConstructor func(fs.FS, string) Extractor

var (
	extractors = map[string]extractorConstructor{}
	lock       sync.RWMutex
)

// RegisterExtractor registers a new extractor, so it can be used by the local storage. The one to be used is
// defined with the configuration option Scanner.Extractor.
func RegisterExtractor(id string, f extractorConstructor) {
	lock.Lock()
	defer lock.Unlock()
	extractors[id] = f
}
