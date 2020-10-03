package metadata

import (
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
	RunSpecs(t, "Metadata Suite")
}
