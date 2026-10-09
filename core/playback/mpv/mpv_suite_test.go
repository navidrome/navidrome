package mpv

import (
	"fmt"
	"os"
	"testing"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const mockMPVEnv = "NAVIDROME_MPV_TEST_ECHO_ARGS"

// TestMain lets the specs run this test binary as a fake mpv that prints its
// arguments one per line, which a shell or batch script can't do portably.
func TestMain(m *testing.M) {
	if os.Getenv(mockMPVEnv) != "" {
		for _, arg := range os.Args {
			fmt.Println(arg)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestMPV(t *testing.T) {
	t.Setenv(mockMPVEnv, "1")
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "MPV Suite")
}
