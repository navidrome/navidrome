package str_test

import (
	"mime"
	"regexp"
	"strings"

	"github.com/navidrome/navidrome/utils/str"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var asciiFilenameRe = regexp.MustCompile(`(?:^|; )filename="([^"]*)"`)

// asciiName returns the quoted filename parameter, the one used by clients that ignore filename*.
func asciiName(filename string) string {
	m := asciiFilenameRe.FindAllStringSubmatch(str.ContentDispositionAttachment(filename), -1)
	ExpectWithOffset(1, m).To(HaveLen(1), "expected exactly one quoted filename parameter")
	return m[0][1]
}

// decodedName returns the name an RFC 6266 client picks, preferring filename* when present.
func decodedName(filename string) string {
	disp, params, err := mime.ParseMediaType(str.ContentDispositionAttachment(filename))
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, disp).To(Equal("attachment"))
	return params["filename"]
}

var _ = Describe("ContentDispositionAttachment", func() {
	Describe("parameter injection", func() {
		const attack = `party"; filename="evil.html.m3u`

		It("does not let a quote in the name open a second parameter", func() {
			Expect(str.ContentDispositionAttachment(attack)).To(Equal(`attachment; filename="party_; filename=_evil.html.m3u"`))
		})

		It("keeps the header parseable", func() {
			Expect(decodedName(attack)).To(Equal("party_; filename=_evil.html.m3u"))
		})

		It("does not let a quote in a non-ASCII name inject either", func() {
			const utf8Attack = `東京"; filename="evil.html.m3u`
			header := str.ContentDispositionAttachment(utf8Attack)
			Expect(strings.Count(header, `"`)).To(Equal(2))
			Expect(decodedName(utf8Attack)).To(Equal("東京_; filename=_evil.html.m3u"))
		})
	})

	Describe("ASCII names", func() {
		It("sends only the quoted filename, unchanged", func() {
			Expect(str.ContentDispositionAttachment("My Playlist.m3u")).To(Equal(`attachment; filename="My Playlist.m3u"`))
		})

		It("keeps leading dots", func() {
			Expect(asciiName("...And Justice for All.zip")).To(Equal("...And Justice for All.zip"))
		})

		It("trims surrounding spaces and trailing dots", func() {
			Expect(asciiName("  Greatest Hits  .zip")).To(Equal("Greatest Hits.zip"))
			Expect(asciiName("Loose End. ")).To(Equal("Loose End"))
		})

		It("falls back to a placeholder when only dots are left", func() {
			Expect(str.ContentDispositionAttachment("...")).To(Equal(`attachment; filename="download"`))
		})

		It("caps a long name whose last dot is not an extension", func() {
			name := asciiName("a." + strings.Repeat("x", 300))
			Expect(len(name)).To(BeNumerically("<=", 255))
			Expect(name).To(HavePrefix("a.xxx"))
		})

		It("replaces path separators and reserved characters", func() {
			Expect(asciiName("AC/DC: Live, 1979?.zip")).To(Equal("AC_DC_ Live_ 1979_.zip"))
		})

		It("drops control characters", func() {
			Expect(asciiName("line\r\nbreak\x00\t.mp3")).To(Equal("linebreak.mp3"))
		})
	})

	Describe("non-ASCII names", func() {
		It("transliterates accents in the ASCII fallback", func() {
			Expect(asciiName("Legião Urbana.zip")).To(Equal("Legiao Urbana.zip"))
		})

		It("converts typographic punctuation in the ASCII fallback", func() {
			Expect(asciiName("She’s a Woman — Live.mp3")).To(Equal("She's a Woman - Live.mp3"))
			Expect(asciiName("“Heroes”.mp3")).To(Equal("_Heroes_.mp3"))
			Expect(decodedName("She’s a Woman.mp3")).To(Equal("She’s a Woman.mp3"))
		})

		It("keeps the extension when no ASCII letters survive", func() {
			Expect(asciiName("東京.mp3")).To(Equal("download.mp3"))
			Expect(asciiName("Кино.m3u")).To(Equal("download.m3u"))
			Expect(asciiName("東京")).To(Equal("download"))
		})

		DescribeTable("filename* carries the sanitized UTF-8 name",
			func(filename, expected string) {
				Expect(decodedName(filename)).To(Equal(expected))
			},
			Entry("accented", "Legião Urbana.zip", "Legião Urbana.zip"),
			Entry("CJK", "東京.zip", "東京.zip"),
			Entry("emoji", "🎵 mix.zip", "🎵 mix.zip"),
			Entry("comma and percent", "Sigur Rós, 100%.zip", "Sigur Rós, 100%.zip"),
			Entry("path separators", "Sigur Rós/Live: Heima.zip", "Sigur Rós_Live_ Heima.zip"),
			Entry("control characters", "Sigur Rós\r\n\x00.zip", "Sigur Rós.zip"),
			Entry("bidi override", "Björk\u202Eexe.mp3", "Björkexe.mp3"),
			Entry("invalid UTF-8", "Bj\xf6rk Café.mp3", "Bj_rk Café.mp3"),
		)

		It("caps the name length and keeps the extension", func() {
			name := decodedName(strings.Repeat("東", 300) + ".zip")
			Expect(len(name)).To(BeNumerically("<=", 255))
			Expect(name).To(HaveSuffix("東.zip"))
		})
	})
})
