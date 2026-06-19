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

	// firstLyric extracts the first StructuredLyric from a LyricsList response.
	firstLyric := func(list *responses.LyricsList) responses.StructuredLyric {
		Expect(list).ToNot(BeNil())
		Expect(list.StructuredLyrics).ToNot(BeEmpty())
		return list.StructuredLyrics[0]
	}

	Describe("getLyricsBySongId (v2 structured)", func() {
		DescribeTable("returns structured lyrics",
			func(title string, wantSynced bool, wantFirstLine, wantLang string) {
				resp := doReq("getLyricsBySongId", "id", songID(title))
				Expect(resp.Status).To(Equal("ok"))
				got := firstLyric(resp.LyricsList)
				Expect(got.Synced).To(Equal(wantSynced))
				Expect(got.Lang).To(Equal(wantLang))
				Expect(got.Line).ToNot(BeEmpty())
				Expect(got.Line[0].Value).To(Equal(wantFirstLine))
			},
			// "xxx" is the ISO 639-2 code for "no language specified"
			Entry("synced LRC embedded", "Embedded Synced LRC", true, "embedded lrc line one", "xxx"),
			Entry("plain text embedded", "Embedded Plain", false, "plain embedded line one", "xxx"),
			Entry("TTML embedded", "Embedded TTML", true, "embedded ttml line", "xxx"),
			Entry("LRC sidecar", "Sidecar LRC", true, "sidecar lrc line", "xxx"),
			Entry("SRT sidecar", "Sidecar SRT", true, "sidecar srt line", "xxx"),
			// YAML sidecar fixture sets "language: eng"; verify Navidrome passes it through unchanged.
			Entry("YAML sidecar", "Sidecar YAML", true, "sidecar yaml line", "eng"),
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
