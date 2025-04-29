package plugins

import (
	"fmt"
	"os/exec"
	"testing"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPlugins(t *testing.T) {
	tests.Init(t, false)
	buildTestPlugins(t, "plugins/testdata")
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugins Suite")
}

func buildTestPlugins(t *testing.T, path string) {
	cwd := path
	fmt.Printf("[BeforeSuite] Current working directory: %s\n", cwd)
	cmd := exec.Command("make", "-C", path)
	out, err := cmd.CombinedOutput()
	fmt.Printf("Make output: %s", string(out))
	if err != nil {
		Fail(fmt.Sprintf("Failed to build test plugins: %v", err))
	}
}
