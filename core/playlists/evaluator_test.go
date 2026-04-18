package playlists_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/navidrome/navidrome/core/playlists"
)

var _ = Describe("SmartPlaylistEvaluator", func() {
	Describe("NoopSmartPlaylistEvaluator", func() {
		It("does not panic when enqueuing", func() {
			noop := playlists.NoopSmartPlaylistEvaluator()
			Expect(func() { noop.Enqueue("some-id") }).ToNot(Panic())
		})
	})
})
