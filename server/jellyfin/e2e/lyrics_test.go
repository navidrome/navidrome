package e2e

import (
	"net/http"

	"github.com/navidrome/navidrome/server/jellyfin/dto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Lyrics", func() {
	BeforeEach(func() { setupTestDB() })

	Describe("PlaybackInfo", func() {
		It("advertises a Lyric stream for a track with embedded lyrics", func() {
			id := songID("Stairway To Heaven")
			var info dto.PlaybackInfoResponse
			parseInto(get("/Items/"+enc(id)+"/PlaybackInfo"), &info)
			var found bool
			for _, s := range info.MediaSources[0].MediaStreams {
				if s.Type == "Lyric" {
					found = true
				}
			}
			Expect(found).To(BeTrue())
		})

		It("does not advertise a Lyric stream for a track without lyrics", func() {
			id := songID("So What")
			var info dto.PlaybackInfoResponse
			parseInto(get("/Items/"+enc(id)+"/PlaybackInfo"), &info)
			for _, s := range info.MediaSources[0].MediaStreams {
				Expect(s.Type).ToNot(Equal("Lyric"))
			}
		})
	})

	Describe("GET /Audio/{id}/Lyrics", func() {
		It("returns the LyricDto for a track with embedded synced lyrics", func() {
			id := songID("Stairway To Heaven")
			var lyrics dto.LyricDto
			parseInto(get("/Audio/"+enc(id)+"/Lyrics"), &lyrics)
			Expect(lyrics.Lyrics).To(HaveLen(2))
			Expect(lyrics.Lyrics[0].Text).To(Equal("There's a lady who's sure"))
			Expect(lyrics.Lyrics[0].Start).ToNot(BeNil())
			Expect(*lyrics.Lyrics[0].Start).To(Equal(int64(10000000)))
			Expect(lyrics.Metadata.IsSynced).To(BeTrue())
		})

		It("returns 404 for a track without lyrics", func() {
			id := songID("So What")
			Expect(get("/Audio/" + enc(id) + "/Lyrics").Code).To(Equal(http.StatusNotFound))
		})

		It("returns 404 for a fabricated id", func() {
			Expect(get("/Audio/" + enc("nope") + "/Lyrics").Code).To(Equal(http.StatusNotFound))
		})
	})

	Describe("HasLyrics badge", func() {
		It("is true for a track with embedded lyrics and omitted/false otherwise", func() {
			var stairway dto.BaseItemDto
			parseInto(get("/Items/"+enc(songID("Stairway To Heaven"))), &stairway)
			Expect(stairway.HasLyrics).To(BeTrue())

			var soWhat dto.BaseItemDto
			parseInto(get("/Items/"+enc(songID("So What"))), &soWhat)
			Expect(soWhat.HasLyrics).To(BeFalse())
		})
	})
})
