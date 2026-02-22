package lyrics

import (
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/gg"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("parseTTML", func() {
	Describe("Multi-language and timing", func() {
		It("should parse multiple language divs with inherited offsets and frame/tick timing", func() {
			content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml" xmlns:ttp="http://www.w3.org/ns/ttml#parameter" ttp:frameRate="30" ttp:subFrameRate="2" ttp:tickRate="10">
  <body>
    <div xml:lang="eng" begin="1s">
      <p begin="2s">Line one</p>
      <p begin="00:00:04:15.1"><span>Line two</span><br/>with break</p>
    </div>
    <div xml:lang="por">
      <p begin="45t">Linha</p>
    </div>
  </body>
</tt>`)

			list, err := parseTTML(content)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(2))

			By("parsing the English track")
			eng := list[0]
			Expect(eng.Lang).To(Equal("eng"))
			Expect(eng.Synced).To(BeTrue())
			Expect(eng.Line[0].Start).To(Equal(gg.P(int64(3000))))
			Expect(eng.Line[0].Value).To(Equal("Line one"))
			Expect(eng.Line[1].Start).To(Equal(gg.P(int64(4517))))
			Expect(eng.Line[1].Value).To(Equal("Line two\nwith break"))

			By("parsing the Portuguese track")
			por := list[1]
			Expect(por.Lang).To(Equal("por"))
			Expect(por.Line[0].Start).To(Equal(gg.P(int64(4500))))
			Expect(por.Line[0].Value).To(Equal("Linha"))
		})
	})

	Describe("Unsupported cue handling", func() {
		It("should skip wallclock cues and keep valid ones", func() {
			content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml">
  <body xml:lang="eng">
    <div>
      <p begin="wallclock(2026-01-01T00:00:00Z)">Skip me</p>
      <p begin="1s">Keep me</p>
    </div>
  </body>
</tt>`)

			list, err := parseTTML(content)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].Line).To(HaveLen(1))
			Expect(list[0].Line[0].Start).To(Equal(gg.P(int64(1000))))
			Expect(list[0].Line[0].Value).To(Equal("Keep me"))
		})
	})

	Describe("Begin/End/Dur with inheritance", func() {
		It("should correctly accumulate nested timing from body, div, and p elements", func() {
			content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml">
  <body xml:lang="eng" begin="10s">
    <div begin="5s" dur="8s">
      <p begin="1s" dur="2s">First line</p>
      <p begin="3s" end="5s">Second line</p>
    </div>
  </body>
</tt>`)

			list, err := parseTTML(content)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].Lang).To(Equal("eng"))
			Expect(list[0].Line).To(HaveLen(2))
			Expect(list[0].Line[0].Start).To(Equal(gg.P(int64(16000))))
			Expect(list[0].Line[0].Value).To(Equal("First line"))
			Expect(list[0].Line[1].Start).To(Equal(gg.P(int64(18000))))
			Expect(list[0].Line[1].Value).To(Equal("Second line"))
		})
	})

	Describe("Non-standard bare second offsets", func() {
		It("should parse bare decimal numbers as seconds", func() {
			content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml">
  <body xml:lang="eng" begin="10">
    <div>
      <p begin="0.170">First line</p>
      <p begin="3.710">Second line</p>
    </div>
  </body>
</tt>`)

			list, err := parseTTML(content)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].Line).To(HaveLen(2))
			Expect(list[0].Line[0].Start).To(Equal(gg.P(int64(10170))))
			Expect(list[0].Line[0].Value).To(Equal("First line"))
			Expect(list[0].Line[1].Start).To(Equal(gg.P(int64(13710))))
			Expect(list[0].Line[1].Value).To(Equal("Second line"))
		})
	})

	Describe("Word timing tokens", func() {
		It("should extract timed tokens from spans including background role", func() {
			content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml" xmlns:ttm="http://www.w3.org/ns/ttml#metadata">
  <body xml:lang="eng">
    <div>
      <p begin="00:01.000" end="00:03.000">
        <span begin="00:01.000" end="00:01.400">He</span><span begin="00:01.400" end="00:01.800">llo</span>
        <span ttm:role="x-bg"><span begin="00:02.000" end="00:02.500">echo</span></span>
      </p>
    </div>
  </body>
</tt>`)

			list, err := parseTTML(content)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].Line).To(HaveLen(1))

			line := list[0].Line[0]
			Expect(line.Start).To(Equal(gg.P(int64(1000))))
			Expect(line.Value).To(Equal("Hello\necho"))
			Expect(line.End).To(Equal(gg.P(int64(3000))))
			Expect(line.Cue).To(HaveLen(3))

			Expect(line.Cue[0]).To(Equal(model.Cue{Start: gg.P(int64(1000)), End: gg.P(int64(1400)), Value: "He"}))
			Expect(line.Cue[1]).To(Equal(model.Cue{Start: gg.P(int64(1400)), End: gg.P(int64(1800)), Value: "llo"}))
			Expect(line.Cue[2]).To(Equal(model.Cue{Start: gg.P(int64(2000)), End: gg.P(int64(2500)), Value: "echo", Role: "x-bg"}))
		})
	})

	Describe("Ambiguous decimal timing", func() {
		It("should prefer absolute timing when values fall inside parent window", func() {
			content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml">
  <body xml:lang="eng">
    <div begin="37.870" end="45.570">
      <p begin="43.444" end="45.570">
        <span begin="43.444" end="43.716">go</span>
        <span begin="43.716" end="43.887">go</span>
      </p>
    </div>
  </body>
</tt>`)

			list, err := parseTTML(content)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].Line).To(HaveLen(1))

			line := list[0].Line[0]
			Expect(line.Start).To(Equal(gg.P(int64(43444))))
			Expect(line.Value).To(Equal("go\ngo"))
			Expect(line.End).To(Equal(gg.P(int64(45570))))
			Expect(line.Cue).To(HaveLen(2))
			Expect(line.Cue[0]).To(Equal(model.Cue{Start: gg.P(int64(43444)), End: gg.P(int64(43716)), Value: "go"}))
			Expect(line.Cue[1]).To(Equal(model.Cue{Start: gg.P(int64(43716)), End: gg.P(int64(43887)), Value: "go"}))
		})
	})

	Describe("Unsynced fallback", func() {
		It("should return unsynced lyrics when no timing is present", func() {
			content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml">
  <body>
    <div>
      <p>No timing here</p>
    </div>
  </body>
</tt>`)

			list, err := parseTTML(content)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].Lang).To(Equal("xxx"))
			Expect(list[0].Synced).To(BeFalse())
			Expect(list[0].Line).To(HaveLen(1))
			Expect(list[0].Line[0].Start).To(BeNil())
			Expect(list[0].Line[0].Value).To(Equal("No timing here"))
		})
	})

	Describe("Metadata tracks", func() {
		It("should produce main, translation, and pronunciation tracks from iTunesMetadata", func() {
			content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml" xmlns:itunes="http://music.apple.com/lyric-ttml-internal">
  <head>
    <metadata>
      <iTunesMetadata xmlns="http://music.apple.com/lyric-ttml-internal">
        <translations>
          <translation xml:lang="es">
            <text for="L1">Hola</text>
            <text for="MISSING">Skip me</text>
          </translation>
        </translations>
        <transliterations>
          <transliteration xml:lang="ja-Latn">
            <text for="L2"><span begin="00:02.000" end="00:02.300" xmlns="http://www.w3.org/ns/ttml">ko</span><span begin="00:02.300" end="00:02.600" xmlns="http://www.w3.org/ns/ttml">nni</span></text>
          </transliteration>
        </transliterations>
      </iTunesMetadata>
    </metadata>
  </head>
  <body xml:lang="ja">
    <div>
      <p begin="00:01.000" end="00:01.500" itunes:key="L1">こんにちは</p>
      <p begin="00:02.000" end="00:02.700" itunes:key="L2">こんばんは</p>
    </div>
  </body>
</tt>`)

			list, err := parseTTML(content)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(3))

			By("checking the main track")
			main := list[0]
			Expect(main.Kind).To(Equal("main"))
			Expect(main.Lang).To(Equal("ja"))
			Expect(main.Line).To(HaveLen(2))

			By("checking the translation track")
			translation := list[1]
			Expect(translation.Kind).To(Equal("translation"))
			Expect(translation.Lang).To(Equal("es"))
			Expect(translation.Line).To(HaveLen(1))
			Expect(translation.Line[0].Start).To(Equal(gg.P(int64(1000))))
			Expect(translation.Line[0].Value).To(Equal("Hola"))
			Expect(translation.Line[0].End).To(Equal(gg.P(int64(1500))))

			By("checking the pronunciation track")
			pronunciation := list[2]
			Expect(pronunciation.Kind).To(Equal("pronunciation"))
			Expect(pronunciation.Lang).To(Equal("ja-latn"))
			Expect(pronunciation.Line).To(HaveLen(1))
			Expect(pronunciation.Line[0].Start).To(Equal(gg.P(int64(2000))))
			Expect(pronunciation.Line[0].Value).To(Equal("konni"))
			Expect(pronunciation.Line[0].End).To(Equal(gg.P(int64(2600))))
			Expect(pronunciation.Line[0].Cue).To(HaveLen(2))
			Expect(pronunciation.Line[0].Cue[0]).To(Equal(model.Cue{Start: gg.P(int64(2000)), End: gg.P(int64(2300)), Value: "ko"}))
			Expect(pronunciation.Line[0].Cue[1]).To(Equal(model.Cue{Start: gg.P(int64(2300)), End: gg.P(int64(2600)), Value: "nni"}))
		})
	})

	Describe("Pronunciation with bare decimal end times", func() {
		It("should correctly parse bare decimal times in transliteration spans", func() {
			content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml" xmlns:itunes="http://music.apple.com/lyric-ttml-internal">
  <head>
    <metadata>
      <iTunesMetadata xmlns="http://music.apple.com/lyric-ttml-internal">
        <transliterations>
          <transliteration xml:lang="ja-Latn">
            <text for="L1"><span begin="2.747" end="3.018" xmlns="http://www.w3.org/ns/ttml">I</span> <span begin="3.018" end="3.179" xmlns="http://www.w3.org/ns/ttml">woke</span> <span begin="3.179" end="3.582" xmlns="http://www.w3.org/ns/ttml">up</span></text>
          </transliteration>
        </transliterations>
      </iTunesMetadata>
    </metadata>
  </head>
  <body xml:lang="ja">
    <div>
      <p begin="00:02.747" end="00:04.000" itunes:key="L1">起きた</p>
    </div>
  </body>
</tt>`)

			list, err := parseTTML(content)
			Expect(err).ToNot(HaveOccurred())

			var pronunciation *model.Lyrics
			for i := range list {
				if list[i].Kind == "pronunciation" {
					pronunciation = &list[i]
					break
				}
			}
			Expect(pronunciation).ToNot(BeNil())
			Expect(pronunciation.Line).To(HaveLen(1))

			line := pronunciation.Line[0]
			Expect(line.Start).To(Equal(gg.P(int64(2747))))
			Expect(line.Value).To(Equal("I woke up"))
			Expect(line.Cue).To(HaveLen(3))
			Expect(line.Cue[0]).To(Equal(model.Cue{Start: gg.P(int64(2747)), End: gg.P(int64(3018)), Value: "I"}))
			Expect(line.Cue[1]).To(Equal(model.Cue{Start: gg.P(int64(3018)), End: gg.P(int64(3179)), Value: "woke"}))
			Expect(line.Cue[2]).To(Equal(model.Cue{Start: gg.P(int64(3179)), End: gg.P(int64(3582)), Value: "up"}))
		})
	})
})
