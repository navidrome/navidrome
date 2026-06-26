package agents

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Song.Equals", func() {
	base := Song{ID: "1", Name: "S", Artist: "A", Artists: []Artist{{ID: "x", Name: "A"}}}
	It("true for identical songs incl Artists", func() {
		Expect(base.Equals(base)).To(BeTrue())
	})
	It("false when Artists differ", func() {
		other := base
		other.Artists = []Artist{{ID: "y", Name: "B"}}
		Expect(base.Equals(other)).To(BeFalse())
	})
	It("false when a scalar differs", func() {
		other := base
		other.Name = "T"
		Expect(base.Equals(other)).To(BeFalse())
	})
	It("true when both have empty Artists and equal scalars", func() {
		a := Song{ID: "1", Name: "S", Artist: "A"}
		Expect(a.Equals(a)).To(BeTrue())
	})
})

var _ = Describe("Song.ArtistList", func() {
	It("returns the Artists slice when present", func() {
		s := Song{Artist: "Primary", ArtistMBID: "mbid-primary", Artists: []Artist{
			{ID: "id-drake", Name: "Drake", MBID: "mbid-drake"},
			{Name: "Future", MBID: "mbid-future"},
		}}
		Expect(s.ArtistList()).To(Equal([]Artist{
			{ID: "id-drake", Name: "Drake", MBID: "mbid-drake"},
			{Name: "Future", MBID: "mbid-future"},
		}))
	})
	It("falls back to the single Artist field with empty ID", func() {
		s := Song{Artist: "Drake", ArtistMBID: "mbid-drake"}
		Expect(s.ArtistList()).To(Equal([]Artist{{Name: "Drake", MBID: "mbid-drake"}}))
	})
	It("returns empty when no artist is set", func() {
		Expect(Song{}.ArtistList()).To(BeEmpty())
	})
})
