package scanner_test

import (
	"context"
	"io/fs"
	"os"
	"testing"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/storage/local"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/metadata"
	"github.com/navidrome/navidrome/tests"
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
	// Only run goleak checks when the GOLEAK env var is set
	if os.Getenv("GOLEAK") != "" {
		// Detect any goroutine leaks in the scanner code under test
		defer goleak.VerifyNone(t,
			goleak.IgnoreTopFunction("github.com/onsi/ginkgo/v2/internal/interrupt_handler.(*InterruptHandler).registerForInterrupts.func2"),
			// The notify library creates internal goroutines for file watching that persist after Stop() is called.
			// These are created by the plugins package tests and are expected behavior.
			goleak.IgnoreTopFunction("github.com/rjeczalik/notify.(*recursiveTree).dispatch"),
		)
	}

	tests.Init(t, true)
	defer db.Close(context.Background())
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scanner Suite")
}
