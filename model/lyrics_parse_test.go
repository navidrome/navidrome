package model

import (
	"strings"

	"github.com/navidrome/navidrome/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

var _ = Describe("ParseLyrics", func() {
	DescribeTable("known suffix routes to the matching parser",
		func(suffix, contents string, wantSynced bool, wantFirst string) {
			list, err := ParseLyrics(GinkgoT().Context(), suffix, "eng", []byte(contents))
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].Synced).To(Equal(wantSynced))
			Expect(list[0].Line[0].Value).To(Equal(wantFirst))
		},
		Entry(".lrc", ".lrc", "[00:01.00]lrc line", true, "lrc line"),
		Entry(".txt plain", ".txt", "plain line", false, "plain line"),
		Entry(".srt", ".srt", "1\n00:00:01,000 --> 00:00:02,000\nsrt line\n", true, "srt line"),
		Entry(".ttml", ".ttml", `<tt xmlns="http://www.w3.org/ns/ttml"><body><div><p begin="00:00.000" end="00:01.000">ttml line</p></div></body></tt>`, true, "ttml line"),
		Entry(".yaml", ".yaml", "version: \"1.0\"\nmetadata:\n  language: eng\nlines:\n  - text: yaml line\n    start_ms: 1000\n", true, "yaml line"),
	)

	It("empty suffix content-sniffs (TTML)", func() {
		ttml := `<tt xmlns="http://www.w3.org/ns/ttml"><body><div><p begin="00:00.000" end="00:01.000">auto ttml</p></div></body></tt>`
		list, err := ParseLyrics(GinkgoT().Context(), "", "eng", []byte(ttml))
		Expect(err).ToNot(HaveOccurred())
		Expect(list).To(HaveLen(1))
		Expect(list[0].Line[0].Value).To(Equal("auto ttml"))
	})

	It("empty suffix content-sniffs (YAML)", func() {
		yaml := "version: \"1.0\"\nmetadata:\n  language: eng\nlines:\n  - text: auto yaml\n    start_ms: 1000\n"
		list, err := ParseLyrics(GinkgoT().Context(), "auto", "eng", []byte(yaml))
		Expect(err).ToNot(HaveOccurred())
		Expect(list).To(HaveLen(1))
		Expect(list[0].Line[0].Value).To(Equal("auto yaml"))
	})

	It("falls back to plain text when a known suffix fails to parse structurally", func() {
		list, err := ParseLyrics(GinkgoT().Context(), ".srt", "eng", []byte("not actually an srt file"))
		Expect(err).ToNot(HaveOccurred())
		Expect(list).To(HaveLen(1))
		Expect(list[0].Synced).To(BeFalse())
		Expect(list[0].Line[0].Value).To(Equal("not actually an srt file"))
	})

	Describe("logging on parser probe failures", func() {
		var hook *test.Hook

		BeforeEach(func() {
			prevLevel := log.CurrentLevel()
			l, h := test.NewNullLogger()
			hook = h
			// Swap the logger before raising the level: SetLevel also forces the
			// current default logger to logrus.TraceLevel, and the null logger would
			// otherwise stay at Info and drop Trace entries before the hook sees them.
			prevLogger := log.SetDefaultLogger(l)
			log.SetLevel(log.LevelTrace)
			DeferCleanup(func() {
				log.SetDefaultLogger(prevLogger)
				log.SetLevel(prevLevel)
			})
		})

		// This is the source of the full-scan log spam: embedded lyrics are parsed
		// with an empty suffix (sniff mode), so every plain-text lyric fails the
		// YAML/SRT/TTML probes on its way to the plain-text fallback. A probe miss
		// during sniffing is expected control flow, not a warning.
		It("logs sniff probe misses at trace only, with file attribution", func() {
			ctx := log.NewContext(GinkgoT().Context(), "file", "/music/song.mp3")
			list, err := ParseLyrics(ctx, "", "eng", []byte("Just a plain\nlyric line\n"))

			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].Line[0].Value).To(Equal("Just a plain"))
			entries := hook.AllEntries()
			Expect(entries).ToNot(BeEmpty(), "probe misses should be observable at trace")
			for _, e := range entries {
				Expect(e.Level).To(Equal(logrus.TraceLevel),
					"sniff-mode probe misses must not be logged above Trace")
				Expect(e.Data).To(HaveKeyWithValue("file", "/music/song.mp3"))
			}
		})

		// A specific suffix means the user declared the format, so a structural
		// failure is worth surfacing loudly — and it must name the file.
		It("warns and names the file when a requested suffix fails to parse", func() {
			ctx := log.NewContext(GinkgoT().Context(), "file", "/music/song.yaml")
			list, err := ParseLyrics(ctx, ".yaml", "eng", []byte("not: [valid, yaml\n"))

			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1)) // still falls back to plain text
			entry := hook.LastEntry()
			Expect(entry).ToNot(BeNil())
			Expect(entry.Level).To(Equal(logrus.WarnLevel))
			Expect(entry.Data).To(HaveKeyWithValue("file", "/music/song.yaml"))
		})
	})
})

var _ = Describe("ParseLyrics content-sniffing", func() {
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

		list, err := ParseLyrics(GinkgoT().Context(), "", "ENG", []byte(content))

		// ParseLyrics's job is to detect TTML and apply the tag language as the
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

		list, err := ParseLyrics(GinkgoT().Context(), "", "eng", []byte(content))

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

		list, err := ParseLyrics(GinkgoT().Context(), "", "POR", []byte(content))

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

		list, err := ParseLyrics(GinkgoT().Context(), "", "eng", []byte(content))

		Expect(err).ToNot(HaveOccurred())
		Expect(list).To(HaveLen(1))
		Expect(list[0].Line).To(Equal([]Line{
			{Start: new(int64(1000)), End: new(int64(2000)), Value: "First subtitle"},
			{Start: new(int64(3000)), End: new(int64(4000)), Value: "Second subtitle"},
		}))
	})

	It("should keep embedded enhanced LRC cues", func() {
		content := "[00:01.00]<00:01.00>Lead <00:01.50>words"

		list, err := ParseLyrics(GinkgoT().Context(), "", "eng", []byte(content))

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

		list, err := ParseLyrics(GinkgoT().Context(), "", "eng", []byte(content))

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

	It("detects a Lyricsfile YAML payload via content-sniffing", func() {
		yaml := "version: \"1.0\"\nmetadata:\n  title: Song\n  language: eng\nlines:\n  - text: sniffed yaml line\n    start_ms: 1000\n"

		list, err := ParseLyrics(GinkgoT().Context(), "", "eng", []byte(yaml))

		Expect(err).ToNot(HaveOccurred())
		Expect(list).To(HaveLen(1))
		Expect(list[0].Synced).To(BeTrue())
		Expect(list[0].Line).To(HaveLen(1))
		Expect(list[0].Line[0].Value).To(Equal("sniffed yaml line"))
	})
})
