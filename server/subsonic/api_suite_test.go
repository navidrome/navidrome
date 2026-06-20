package subsonic

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

func TestSubsonicApi(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Subsonic API Suite")
}

// newLocalStorage fatals if the default extractor is not registered.
// Register a no-op so storage.For works in sidecar-lyrics tests.
var _ = BeforeSuite(func() {
	local.RegisterExtractor(consts.DefaultScannerExtractor, func(fs.FS, string) local.Extractor {
		return &subsonicNoopExtractor{}
	})
})

type subsonicNoopExtractor struct{}

func (e *subsonicNoopExtractor) Parse(_ ...string) (map[string]metadata.Info, error) { return nil, nil }
func (e *subsonicNoopExtractor) Version() string                                     { return "noop" }
