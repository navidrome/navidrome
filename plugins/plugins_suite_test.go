package plugins

import (
	"os"
	"os/exec"
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
	wasmPath := "plugins/testdata/fake_artist_agent/plugin.wasm"
	_ = os.Remove(wasmPath)
	cmd := exec.Command("go", "build", "-buildmode=c-shared", "-o", wasmPath, "./plugins/testdata/fake_artist_agent")
	cmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	Expect(cmd.Run()).To(Succeed())
	Expect(wasmPath).To(BeAnExistingFile())

	albumWasmPath := "plugins/testdata/fake_album_agent/plugin.wasm"
	_ = os.Remove(albumWasmPath)
	cmd = exec.Command("go", "build", "-buildmode=c-shared", "-o", albumWasmPath, "./plugins/testdata/fake_album_agent")
	cmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	Expect(cmd.Run()).To(Succeed())
	Expect(albumWasmPath).To(BeAnExistingFile())
})
