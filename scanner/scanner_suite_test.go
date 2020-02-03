package scanner

import (
	"os"
	"strings"
	"testing"

	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/tests"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestScanner(t *testing.T) {
	tests.Init(t, true)
	log.SetLevel(log.LevelCritical)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scanner Suite")
}

func P(path string) string {
	return strings.ReplaceAll(path, "/", string(os.PathSeparator))
}
