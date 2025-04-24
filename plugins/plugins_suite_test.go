package plugins

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPlugins(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugins Suite")
}

var _ = BeforeSuite(func() {
	cwd, _ := os.Getwd()
	fmt.Printf("[BeforeSuite] Current working directory: %s\n", cwd)
	absPath, err := filepath.Abs("./testdata")
	Expect(err).To(BeNil())
	cmd := exec.Command("make", "-C", absPath)
	out, err := cmd.CombinedOutput()
	Expect(err).To(BeNil(), "make output:\n%s", string(out))
})
