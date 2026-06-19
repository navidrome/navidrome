package e2e

import (
	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Lyrics endpoints", func() {
	BeforeEach(func() {
		setupTestDB()
	})

	// songID resolves a track title to its Subsonic ID via search3.
	songID := func(title string) string {
		resp := doReq("search3", "query", title, "songCount", "1", "artistCount", "0", "albumCount", "0")
		Expect(resp.Status).To(Equal("ok"))
		Expect(resp.SearchResult3).ToNot(BeNil())
		Expect(resp.SearchResult3.Song).ToNot(BeEmpty(), "expected to find song %q", title)
		return resp.SearchResult3.Song[0].Id
	}

	// main extracts the first StructuredLyric from a LyricsList response.
	main := func(list *responses.LyricsList) responses.StructuredLyric {
		Expect(list).ToNot(BeNil())
		Expect(list.StructuredLyrics).ToNot(BeEmpty())
		return list.StructuredLyrics[0]
	}

	Describe("getLyricsBySongId (v2 structured)", func() {
		DescribeTable("returns structured lyrics for embedded formats",
			func(title string, wantSynced bool, wantFirstLine string) {
				resp := doReq("getLyricsBySongId", "id", songID(title))
				Expect(resp.Status).To(Equal("ok"))
				got := main(resp.LyricsList)
				Expect(got.Synced).To(Equal(wantSynced))
				Expect(got.Line).ToNot(BeEmpty())
				Expect(got.Line[0].Value).To(Equal(wantFirstLine))
			},
			Entry("synced LRC", "Embedded Synced LRC", true, "embedded lrc line one"),
			Entry("plain text", "Embedded Plain", false, "plain embedded line one"),
			Entry("TTML", "Embedded TTML", true, "embedded ttml line"),
		)

		DescribeTable("returns structured lyrics for sidecar formats",
			func(title string, wantSynced bool, wantFirstLine string) {
				resp := doReq("getLyricsBySongId", "id", songID(title))
				Expect(resp.Status).To(Equal("ok"))
				got := main(resp.LyricsList)
				Expect(got.Synced).To(Equal(wantSynced))
				Expect(got.Line).ToNot(BeEmpty())
				Expect(got.Line[0].Value).To(Equal(wantFirstLine))
			},
			Entry("LRC sidecar", "Sidecar LRC", true, "sidecar lrc line"),
			Entry("SRT sidecar", "Sidecar SRT", true, "sidecar srt line"),
			Entry("YAML sidecar", "Sidecar YAML", true, "sidecar yaml line"),
		)
	})

	Describe("getLyrics (legacy artist/title)", func() {
		It("returns the main lyric as plain text for an embedded match", func() {
			resp := doReq("getLyrics", "artist", "Lyric Tester", "title", "Embedded Plain")
			Expect(resp.Status).To(Equal("ok"))
			Expect(resp.Lyrics).ToNot(BeNil())
			Expect(resp.Lyrics.Value).To(ContainSubstring("plain embedded line one"))
			Expect(resp.Lyrics.Value).To(ContainSubstring("plain embedded line two"))
		})
	})
})
