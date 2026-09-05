package str_test

import (
	"bufio"
	"mime"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"

	"github.com/navidrome/navidrome/utils/str"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var asciiFilenameRe = regexp.MustCompile(`(?:^|; )filename="([^"]*)"`)

// asciiName returns the plain (quoted, ASCII) filename parameter, the one used by
// clients that don't understand filename*.
func asciiName(filename string) string {
	m := asciiFilenameRe.FindAllStringSubmatch(str.ContentDispositionAttachment(filename), -1)
	ExpectWithOffset(1, m).To(HaveLen(1), "expected exactly one quoted filename parameter")
	return m[0][1]
}

// decodedName returns what a client sees after parsing the header, which per
// RFC 6266 §4.3 means the filename* value wins when both are present.
func decodedName(filename string) string {
	disp, params, err := mime.ParseMediaType(str.ContentDispositionAttachment(filename))
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, disp).To(Equal("attachment"))
	return params["filename"]
}

var _ = Describe("ContentDispositionAttachment", func() {
	Describe("parameter injection", func() {
		// Without escaping, this name yields:
		//   attachment; filename="party"; filename="evil.html.m3u"
		const attack = `party"; filename="evil.html.m3u`

		It("does not let a quote in the name open a second parameter", func() {
			Expect(asciiName(attack)).To(Equal("party_; filename=_evil.html.m3u"))
		})

		It("keeps the header parseable, with the attack contained in the value", func() {
			Expect(decodedName(attack)).To(Equal(attack))
		})

		It("emits no quotes beyond the ones delimiting the ASCII name", func() {
			Expect(strings.Count(str.ContentDispositionAttachment(attack), `"`)).To(Equal(2))
		})

		It("survives a round trip through net/http", func() {
			rec := httptest.NewRecorder()
			rec.Header().Set("Content-Disposition", str.ContentDispositionAttachment(`x"; filename="y.exe`))
			rec.WriteHeader(http.StatusOK)

			var sb strings.Builder
			w := bufio.NewWriter(&sb)
			Expect(rec.Result().Write(w)).To(Succeed())
			Expect(w.Flush()).To(Succeed())

			Expect(sb.String()).ToNot(ContainSubstring(`filename="y.exe"`))
		})
	})

	Describe("ASCII filename parameter", func() {
		It("leaves a name that is already safe untouched", func() {
			Expect(asciiName("My Playlist.m3u")).To(Equal("My Playlist.m3u"))
		})

		It("strips path separators and control characters", func() {
			Expect(asciiName("../../etc/passwd\x00\t")).To(Equal("_.._etc_passwd"))
		})

		It("replaces commas, which some clients treat as value separators", func() {
			Expect(asciiName("Bowie, David.zip")).To(Equal("Bowie_ David.zip"))
		})

		It("drops non-ASCII runes", func() {
			Expect(asciiName("Legião Urbana.zip")).To(Equal("Legio Urbana.zip"))
		})

		It("falls back to a placeholder when nothing ASCII survives", func() {
			Expect(asciiName("東京")).To(Equal("download"))
		})

		It("trims leading and trailing dots and spaces", func() {
			Expect(asciiName("  ..hidden..  ")).To(Equal("hidden"))
		})
	})

	Describe("filename* parameter", func() {
		DescribeTable("round-trips the original name",
			func(filename string) {
				Expect(decodedName(filename)).To(Equal(filename))
			},
			Entry("plain ASCII", "My Playlist.m3u"),
			Entry("accented", "Legião Urbana.zip"),
			Entry("CJK", "東京.zip"),
			Entry("emoji", "🎵 mix.zip"),
			Entry("comma", "Bowie, David.zip"),
			Entry("quote", `12" Mixes.zip`),
			Entry("percent", "100% Hits.zip"),
			Entry("semicolon", "a;b.zip"),
		)
	})
})
