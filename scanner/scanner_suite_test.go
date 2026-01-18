package scanner_test

import (
	"context"
	"os"
	"testing"

	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/goleak"
)

func TestScanner(t *testing.T) {
	// Only run goleak checks when not in CI environment
	if os.Getenv("CI") == "" {
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
