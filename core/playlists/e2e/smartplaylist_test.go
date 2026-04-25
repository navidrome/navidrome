package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Smart Playlists", func() {
	BeforeEach(func() {
		setupTestDB()
	})

	Describe("String fields", func() {
		It("matches by exact title", func() {
			results := evaluateRule(`{"all":[{"is":{"title":"Something"}}]}`)
			Expect(results).To(ConsistOf("Something"))
		})

		It("matches by title contains", func() {
			results := evaluateRule(`{"all":[{"contains":{"title":"the"}}]}`)
			Expect(results).To(ConsistOf("Come Together", "All Along the Watchtower", "We Are the Champions"))
		})

		It("matches by artist startsWith", func() {
			results := evaluateRule(`{"all":[{"startsWith":{"artist":"Led"}}]}`)
			Expect(results).To(ConsistOf("Stairway To Heaven", "Black Dog"))
		})
	})
})
