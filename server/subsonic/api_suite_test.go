package subsonic

import (
	"testing"

	"github.com/navidrome/navidrome/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSubsonicApi(t *testing.T) {
	log.SetLevel(log.LevelCritical)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Subsonic API Suite")
}
