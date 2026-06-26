package agents

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

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
