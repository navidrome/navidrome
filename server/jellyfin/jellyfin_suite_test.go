package jellyfin

import (
	"net/http"
	"testing"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestJellyfinApi(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Jellyfin API Suite")
}

// testID maps a readable label to a deterministic canonical id, so fixtures exercise the same
// id shape production uses.
func testID(label string) string { return id.NewHash("jellyfin-test", label) }

// invoke runs a handler through normalizeQueryKeys, mirroring the router. These unit tests call
// handlers directly (with withChiURLParam for path params) instead of routing, so without this the
// case-insensitive query folding real requests get would be skipped and PascalCase params dropped.
func invoke(h http.HandlerFunc, w http.ResponseWriter, r *http.Request) {
	normalizeQueryKeys(h).ServeHTTP(w, r)
}
