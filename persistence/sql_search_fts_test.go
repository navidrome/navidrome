package persistence

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("buildFTS5Query", func() {
	It("returns empty string for empty input", func() {
		Expect(buildFTS5Query("")).To(BeEmpty())
	})

	It("returns empty string for whitespace-only input", func() {
		Expect(buildFTS5Query("   ")).To(BeEmpty())
	})

	It("passes through a single word", func() {
		Expect(buildFTS5Query("beatles")).To(Equal("beatles"))
	})

	It("joins multiple words with implicit AND", func() {
		Expect(buildFTS5Query("abbey road")).To(Equal("abbey road"))
	})

	It("preserves quoted phrases", func() {
		Expect(buildFTS5Query(`"the beatles"`)).To(Equal(`"the beatles"`))
	})

	It("preserves prefix wildcard", func() {
		Expect(buildFTS5Query("beat*")).To(Equal("beat*"))
	})

	It("strips FTS5 operators to prevent injection", func() {
		Expect(buildFTS5Query("AND OR NOT NEAR")).To(Equal("and or not near"))
	})

	It("strips special FTS5 syntax characters", func() {
		Expect(buildFTS5Query("test^col:val")).To(Equal("test col val"))
	})

	It("handles mixed phrases and words", func() {
		Expect(buildFTS5Query(`"the beatles" abbey`)).To(Equal(`"the beatles" abbey`))
	})

	It("handles prefix with multiple words", func() {
		Expect(buildFTS5Query("beat* abbey")).To(Equal("beat* abbey"))
	})

	It("collapses multiple spaces", func() {
		Expect(buildFTS5Query("abbey   road")).To(Equal("abbey road"))
	})
})
