package podcasts_test

import (
	"testing"

	"github.com/navidrome/navidrome/core/podcasts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPodcasts(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	// This suite's specs fetch RSS feeds/episodes from an httptest.Server, which
	// always binds to loopback - safeHTTPTransport's SSRF guard would otherwise
	// refuse every request the suite makes. See AllowLoopbackHTTPForTests's own
	// doc comment: every other reserved/private/link-local address is still
	// refused, so this doesn't disable the guard, just narrows it for this run.
	podcasts.AllowLoopbackHTTPForTests()
	RegisterFailHandler(Fail)
	RunSpecs(t, "Podcasts Suite")
}
