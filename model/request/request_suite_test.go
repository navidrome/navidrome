package request

import (
	"testing"

	"github.com/navidrome/navidrome/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// tests.Init is not used here: the tests package imports model/request, so importing it
// back would create an import cycle.
func TestRequest(t *testing.T) {
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Request Suite")
}
