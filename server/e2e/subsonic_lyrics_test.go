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

	// songLyrics extension v1: getLyricsBySongId without the enhanced parameter
	// returns line-level structured lyrics (line[], lang, synced) and must NOT
	// emit any v2/enhanced fields — no cueLine, kind, or agents — even for
	// formats that carry word-level timing (ELRC, Lyricsfile YAML).
	Describe("getLyricsBySongId v1 (line-level, not enhanced)", func() {
		DescribeTable("returns line-level lyrics without enhanced fields",
			func(title string, wantSynced bool, wantLang string) {
				resp := doReq("getLyricsBySongId", "id", songID(title))
				Expect(resp.Status).To(Equal("ok"))
				got := firstLyric(resp.LyricsList)

				Expect(got.Synced).To(Equal(wantSynced))
				Expect(got.Lang).To(Equal(wantLang))
				Expect(got.Line).ToNot(BeEmpty())
				Expect(got.Line[0].Value).To(Equal(firstFixtureLine))

				// v1 must not expose any enhanced (v2) data.
				Expect(got.CueLine).To(BeEmpty())
				Expect(got.Kind).To(BeEmpty())
				Expect(got.Agents).To(BeEmpty())
			},
			// "xxx" is the ISO 639-2 code for "no language specified"; the .lrc/.elrc
			// fixtures declare [lang:eng], the .ttml declares xml:lang, the .yaml sets
			// language: eng, while .srt carries no language and the embedded plain
			// text has none — so each format exercises a different language path.
			Entry("embedded enhanced LRC (word-level)", "Embedded Enhanced LRC", true, "eng"),
			Entry("embedded plain text", "Embedded Plain", false, "xxx"),
			Entry("embedded TTML", "Embedded TTML", true, "eng"),
			Entry("LRC sidecar", "Sidecar LRC", true, "eng"),
			Entry("SRT sidecar", "Sidecar SRT", true, "xxx"),
			Entry("YAML sidecar (word-level)", "Sidecar YAML", true, "eng"),
		)
	})

	// songLyrics extension v2: getLyricsBySongId?enhanced=true opts in to
	// word/syllable-level timing (cueLine) and the kind classification. Every
	// format gains kind="main" for a single untyped lyric layer; only formats
	// that carry word-level timing (ELRC, TTML word spans, Lyricsfile YAML)
	// surface a cueLine. Line-level formats (LRC, SRT, plain) still yield none.
	Describe("getLyricsBySongId v2 (enhanced)", func() {
		DescribeTable("returns enhanced lyrics, with cueLine only for word-level sources",
			func(title string, wantCueLine bool) {
				resp := doReq("getLyricsBySongId", "id", songID(title), "enhanced", "true")
				Expect(resp.Status).To(Equal("ok"))
				got := firstLyric(resp.LyricsList)

				Expect(got.Kind).To(Equal("main"))
				if wantCueLine {
					Expect(got.CueLine).ToNot(BeEmpty())
					// The first line has one cue per word: "Should auld acquaintance be forgot,".
					Expect(got.CueLine[0].Cue).To(HaveLen(5))
					Expect(got.CueLine[0].Cue[0].Value).To(Equal("Should "))
				} else {
					Expect(got.CueLine).To(BeEmpty())
					Expect(got.Line).ToNot(BeEmpty())
				}
			},
			Entry("embedded enhanced LRC (word-level)", "Embedded Enhanced LRC", true),
			Entry("embedded TTML (word-level spans)", "Embedded TTML", true),
			Entry("YAML sidecar (word-level)", "Sidecar YAML", true),
			Entry("embedded plain text (no timing)", "Embedded Plain", false),
			Entry("LRC sidecar (line-level)", "Sidecar LRC", false),
			Entry("SRT sidecar (line-level)", "Sidecar SRT", false),
		)
	})

	// getLyrics is the original Subsonic (pre-OpenSubsonic) endpoint. It looks up
	// by artist/title and returns the main lyric flattened to plain text — every
	// line's Value joined by newlines, with all timing/markup dropped. Synced and
	// word-level formats (ELRC/TTML/SRT/YAML) all degrade to plain text here.
	Describe("getLyrics (legacy artist/title)", func() {
		DescribeTable("returns the main lyric as plain text across formats and sources",
			func(title string) {
				resp := doReq("getLyrics", "artist", "Lyric Tester", "title", title)
				Expect(resp.Status).To(Equal("ok"))
				Expect(resp.Lyrics).ToNot(BeNil())
				Expect(resp.Lyrics.Artist).To(Equal("Lyric Tester"))
				Expect(resp.Lyrics.Title).To(Equal(title))
				Expect(resp.Lyrics.Value).To(ContainSubstring(firstFixtureLine))

				// No timing markup leaks into the plain-text value, regardless of the
				// source format: no LRC brackets/word markers, SRT arrows, or XML tags.
				Expect(resp.Lyrics.Value).ToNot(ContainSubstring("["))
				Expect(resp.Lyrics.Value).ToNot(ContainSubstring("-->"))
				Expect(resp.Lyrics.Value).ToNot(ContainSubstring("<"))
			},
			Entry("embedded enhanced LRC", "Embedded Enhanced LRC"),
			Entry("embedded plain text", "Embedded Plain"),
			Entry("embedded TTML", "Embedded TTML"),
			Entry("LRC sidecar", "Sidecar LRC"),
			Entry("SRT sidecar", "Sidecar SRT"),
			Entry("YAML sidecar", "Sidecar YAML"),
		)
	})
})
