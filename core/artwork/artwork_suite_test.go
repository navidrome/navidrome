package artwork

import (
	"io/fs"
	"net/url"
	"os"
	"testing"

	"github.com/navidrome/navidrome/core/storage"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/metadata"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestArtwork(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Artwork Suite")
}

// osDirFS wraps os.DirFS as a storage.MusicFS for integration tests.
// ReadTags is not used by albumArtworkReader, so it is left as a stub.
type osDirFS struct{ fs.FS }

func (o osDirFS) ReadTags(...string) (map[string]metadata.Info, error) { return nil, nil }

func init() {
	// Register a "testfile" storage scheme that creates an os.DirFS-backed MusicFS.
	// Used by artwork integration tests that need real files but not the taglib extractor.
	storage.Register("testfile", func(u url.URL) storage.Storage {
		return &osDirStorage{root: u.Path}
	})
}

type osDirStorage struct{ root string }

func (s *osDirStorage) FS() (storage.MusicFS, error) {
	if _, err := os.Stat(s.root); err != nil {
		return nil, err
	}
	return osDirFS{os.DirFS(s.root)}, nil
}
