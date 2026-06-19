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
		// The legacy endpoint flattens the main lyric to plain text: it emits each
		// line's Value joined by newlines, dropping all timing/markup. This is the
		// v1 contract — synced structured formats (LRC/TTML/SRT/YAML) all "fall
		// back" to LRC-style plain text here.
		DescribeTable("returns the main lyric as plain text across formats and sources",
			func(title string, wantLines []string) {
				resp := doReq("getLyrics", "artist", "Lyric Tester", "title", title)
				Expect(resp.Status).To(Equal("ok"))
				Expect(resp.Lyrics).ToNot(BeNil())
				Expect(resp.Lyrics.Artist).To(Equal("Lyric Tester"))
				Expect(resp.Lyrics.Title).To(Equal(title))
				for _, line := range wantLines {
					Expect(resp.Lyrics.Value).To(ContainSubstring(line))
				}
				// v1 fallback: no timing markup leaks into the plain-text value,
				// regardless of the source format (LRC brackets, SRT arrows, XML
				// tags). The structured content is reduced to LRC-style plain text.
				Expect(resp.Lyrics.Value).ToNot(ContainSubstring("["))
				Expect(resp.Lyrics.Value).ToNot(ContainSubstring("-->"))
				Expect(resp.Lyrics.Value).ToNot(ContainSubstring("<"))
			},
			Entry("embedded synced LRC", "Embedded Synced LRC", []string{"embedded lrc line one", "embedded lrc line two"}),
			Entry("embedded plain text", "Embedded Plain", []string{"plain embedded line one", "plain embedded line two"}),
			Entry("embedded TTML", "Embedded TTML", []string{"embedded ttml line"}),
			Entry("LRC sidecar", "Sidecar LRC", []string{"sidecar lrc line"}),
			Entry("SRT sidecar", "Sidecar SRT", []string{"sidecar srt line"}),
			Entry("YAML sidecar", "Sidecar YAML", []string{"sidecar yaml line"}),
		)
	})
})
