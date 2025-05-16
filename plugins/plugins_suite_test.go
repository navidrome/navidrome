package plugins

import (
	"os/exec"
	"testing"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const testDataDir = "plugins/testdata"

func TestPlugins(t *testing.T) {
	tests.Init(t, false)
	buildTestPlugins(t, testDataDir)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugins Suite")
}

func buildTestPlugins(t *testing.T, path string) {
	t.Helper()
	t.Logf("[BeforeSuite] Current working directory: %s", path)
	cmd := exec.Command("make", "-C", path)
	out, err := cmd.CombinedOutput()
	t.Logf("[BeforeSuite] Make output: %s", string(out))
	if err != nil {
		t.Fatalf("Failed to build test plugins: %v", err)
	}
}
