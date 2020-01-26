package scanner

import (
	"testing"

	"github.com/deluan/navidrome/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// TODO Fix OS dependencies
func xTestScanner(t *testing.T) {
	log.SetLevel(log.LevelCritical)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scanner Suite")
}
