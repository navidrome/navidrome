package dto

import (
	"testing"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDto(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Jellyfin DTO Suite")
}

// testID maps a readable label to a deterministic canonical id, so fixtures exercise the same
// id shape production uses.
func testID(label string) string { return id.NewHash("jellyfin-test", label) }
