package engine

import (
	"testing"

	"github.com/deluan/navidrome/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestEngine(t *testing.T) {
	log.SetLevel(log.LevelCritical)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Engine Suite")
}
