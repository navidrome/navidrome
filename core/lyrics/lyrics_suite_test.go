package lyrics_test

import (
	"io/fs"
	"testing"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/storage/local"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/metadata"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLyrics(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Lyrics Suite")
}

// Register a no-op extractor so that storage.For("file://...") works in tests.
// The lyrics package only needs FS() for sidecar reads; ReadTags is never called.
var _ = BeforeSuite(func() {
	local.RegisterExtractor(consts.DefaultScannerExtractor, func(fs.FS, string) local.Extractor {
		return &noopExtractor{}
	})
})

type noopExtractor struct{}

func (e *noopExtractor) Parse(_ ...string) (map[string]metadata.Info, error) { return nil, nil }
func (e *noopExtractor) Version() string                                     { return "noop" }
