//go:build windows

package plugins

import (
	"testing"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Runs the subset of plugin specs compiled on Windows (files without the
// //go:build !windows tag): capabilities, manager_cache, manager_plugin,
// manifest, package. WASM-runtime-dependent specs live in !windows-tagged
// files and aren't reached here.
func TestPlugins(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugins Suite")
}
