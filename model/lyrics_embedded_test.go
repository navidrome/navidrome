package model

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParseEmbedded", func() {
	It("should parse embedded TTML with the tag language as the default", func() {
		content := `<tt xmlns="http://www.w3.org/ns/ttml" xmlns:ttm="http://www.w3.org/ns/ttml#metadata">
  <head>
    <metadata>
      <ttm:agent xml:id="lead" ttm:type="person">
        <ttm:name>Lead Vocal</ttm:name>
      </ttm:agent>
    </metadata>
  </head>
  <body>
    <div>
      <p begin="00:00:01.000" end="00:00:03.000">
        <span begin="00:00:01.000" end="00:00:02.000" ttm:agent="lead">Hello </span><span begin="00:00:02.000" end="00:00:03.000" ttm:agent="lead">world</span>
      </p>
    </div>
  </body>
</tt>`

		list, err := ParseEmbedded("ENG", content)

		// ParseEmbedded's job is to detect TTML and apply the tag language as the
		// default; the parser's cue/agent details are covered in lyrics_ttml_test.go.
		Expect(err).ToNot(HaveOccurred())
		Expect(list).To(HaveLen(1))
		Expect(list[0].Kind).To(Equal("main"))
		Expect(list[0].Lang).To(Equal("eng"))
		Expect(list[0].Synced).To(BeTrue())
		Expect(list[0].Line[0].Value).To(Equal("Hello world"))
	})

	It("should preserve embedded TTML translation and pronunciation tracks", func() {
		content := `<tt xmlns="http://www.w3.org/ns/ttml" xmlns:itunes="http://music.apple.com/lyric-ttml-internal">
  <head>
    <metadata>
      <iTunesMetadata xmlns="http://music.apple.com/lyric-ttml-internal">
        <translations>
          <translation xml:lang="es">
            <text for="L1">Hola</text>
          </translation>
        </translations>
        <transliterations>
          <transliteration xml:lang="ja-Latn">
            <text for="L1"><span begin="00:00:01.000" end="00:00:01.300" xmlns="http://www.w3.org/ns/ttml">ko</span><span begin="00:00:01.300" end="00:00:01.600" xmlns="http://www.w3.org/ns/ttml">nni</span></text>
          </transliteration>
        </transliterations>
      </iTunesMetadata>
    </metadata>
  </head>
  <body xml:lang="ja">
    <div>
      <p begin="00:00:01.000" end="00:00:02.000" itunes:key="L1">こんにちは</p>
    </div>
  </body>
</tt>`

		list, err := ParseEmbedded("eng", content)

		Expect(err).ToNot(HaveOccurred())
		Expect(list).To(HaveLen(3))
		Expect(list[0].Kind).To(Equal("main"))
		Expect(list[0].Lang).To(Equal("ja"))
		Expect(list[0].Line[0].Value).To(Equal("こんにちは"))
		Expect(list[1].Kind).To(Equal("translation"))
		Expect(list[1].Lang).To(Equal("es"))
		Expect(list[1].Line[0].Value).To(Equal("Hola"))
		Expect(list[2].Kind).To(Equal("pronunciation"))
		Expect(list[2].Lang).To(Equal("ja-latn"))
		Expect(list[2].Line[0].Value).To(Equal("konni"))
		Expect(list[2].Line[0].Cue).To(HaveLen(2))
	})

	It("should parse embedded SRT with the tag language", func() {
		content := `1
00:00:18,800 --> 00:00:22,800
We're from subtitles

2
00:00:22,801 --> 00:00:26,000
Another subtitle line`

		list, err := ParseEmbedded("POR", content)

		Expect(err).ToNot(HaveOccurred())
		Expect(list).To(Equal(LyricList{
			{
				Lang: "por",
				Line: []Line{
					{
						Start: new(int64(18800)),
						End:   new(int64(22800)),
						Value: "We're from subtitles",
					},
					{
						Start: new(int64(22801)),
						End:   new(int64(26000)),
						Value: "Another subtitle line",
					},
				},
				Synced: true,
			},
		}))
	})

	It("should parse embedded SRT blocks separated by whitespace-only blank lines", func() {
		content := "1\n00:00:01,000 --> 00:00:02,000\nFirst subtitle\n   \n2\n00:00:03,000 --> 00:00:04,000\nSecond subtitle"

		list, err := ParseEmbedded("eng", content)

		Expect(err).ToNot(HaveOccurred())
		Expect(list).To(HaveLen(1))
		Expect(list[0].Line).To(Equal([]Line{
			{Start: new(int64(1000)), End: new(int64(2000)), Value: "First subtitle"},
			{Start: new(int64(3000)), End: new(int64(4000)), Value: "Second subtitle"},
		}))
	})

	It("should keep embedded enhanced LRC cues", func() {
		content := "[00:01.00]<00:01.00>Lead <00:01.50>words"

		list, err := ParseEmbedded("eng", content)

		Expect(err).ToNot(HaveOccurred())
		Expect(list).To(HaveLen(1))
		Expect(list[0].Lang).To(Equal("eng"))
		Expect(list[0].Synced).To(BeTrue())
		Expect(list[0].Line[0].Value).To(Equal("Lead words"))
		Expect(list[0].Line[0].Cue).To(HaveLen(2))
	})

	It("should fall back to plain lyrics when embedded TTML is invalid", func() {
		content := `<tt xmlns="http://www.w3.org/ns/ttml">
  <body>
    <p begin="not-a-time">Broken</p>
  </body>
</tt>`

		list, err := ParseEmbedded("eng", content)

		Expect(err).ToNot(HaveOccurred())
		Expect(list).To(HaveLen(1))
		Expect(list[0].Lang).To(Equal("eng"))
		Expect(list[0].Synced).To(BeFalse())
		Expect(list[0].Line).ToNot(BeEmpty())
		values := make([]string, 0, len(list[0].Line))
		for _, line := range list[0].Line {
			values = append(values, line.Value)
		}
		Expect(strings.Join(values, "\n")).To(ContainSubstring("Broken"))
	})
})
