package dto

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParseFields", func() {
	It("parses a single comma-separated value", func() {
		f := ParseFields("Genres,MediaSources")
		Expect(f.Has("Genres")).To(BeTrue())
		Expect(f.Has("MediaSources")).To(BeTrue())
	})

	It("parses fields spread across repeated params", func() {
		f := ParseFields("Genres", "MediaSources", "SortName")
		Expect(f.Has("Genres")).To(BeTrue())
		Expect(f.Has("MediaSources")).To(BeTrue())
		Expect(f.Has("SortName")).To(BeTrue())
	})

	It("returns an empty set for no values", func() {
		Expect(ParseFields()).To(BeEmpty())
		Expect(ParseFields("")).To(BeEmpty())
	})
})
