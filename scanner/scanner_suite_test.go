package scanner_test

import (
	"context"
	"io/fs"
	"testing"

	"github.com/SunilSherkhane/navidrome/consts"
	"github.com/SunilSherkhane/navidrome/core/storage/local"
	"github.com/SunilSherkhane/navidrome/db"
	"github.com/SunilSherkhane/navidrome/log"
	"github.com/SunilSherkhane/navidrome/model/metadata"
	"github.com/SunilSherkhane/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/goleak"
)

// The local storage is registered in this test binary, so any spec (or background watcher)
// touching a file:// library needs a default extractor to avoid a startup fatal.
type noopSuiteExtractor struct{}

func (noopSuiteExtractor) Parse(...string) (map[string]metadata.Info, error) { return nil, nil }
func (noopSuiteExtractor) Version() string                                   { return "0" }

func init() {
	local.RegisterExtractor(consts.DefaultScannerExtractor, func(fs.FS, string) local.Extractor {
		return noopSuiteExtractor{}
	})
}

func TestScanner(t *testing.T) {
	// Detect any goroutine leaks in the scanner code under test
	defer goleak.VerifyNone(t,
		goleak.IgnoreTopFunction("github.com/onsi/ginkgo/v2/internal/interrupt_handler.(*InterruptHandler).registerForInterrupts.func2"),
		// The notify library keeps internal goroutines alive after Stop(). The backend picks the tree per
		// platform: recursive on macOS (FSEvents), nonrecursive on Linux (inotify), so ignore both.
		goleak.IgnoreTopFunction("github.com/rjeczalik/notify.(*recursiveTree).dispatch"),
		goleak.IgnoreTopFunction("github.com/rjeczalik/notify.(*nonrecursiveTree).dispatch"),
		goleak.IgnoreTopFunction("github.com/rjeczalik/notify.(*nonrecursiveTree).internal"),
	)

	tests.Init(t, true)
	defer db.Close(context.Background())
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scanner Suite")
}
