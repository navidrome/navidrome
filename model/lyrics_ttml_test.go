package model

import (
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

			list, err := parseTTML("xxx", content)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(2))

			By("parsing the English track")
			eng := list[0]
			Expect(eng.Lang).To(Equal("eng"))
			Expect(eng.Synced).To(BeTrue())
			Expect(eng.Line[0].Start).To(Equal(new(int64(3000))))
			Expect(eng.Line[0].Value).To(Equal("Line one"))
			Expect(eng.Line[1].Start).To(Equal(new(int64(4517))))
			Expect(eng.Line[1].Value).To(Equal("Line two\nwith break"))

			By("parsing the Portuguese track")
			por := list[1]
			Expect(por.Lang).To(Equal("por"))
			Expect(por.Line[0].Start).To(Equal(new(int64(4500))))
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

			list, err := parseTTML("xxx", content)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].Line).To(HaveLen(1))
			Expect(list[0].Line[0].Start).To(Equal(new(int64(1000))))
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

			list, err := parseTTML("xxx", content)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].Lang).To(Equal("eng"))
			Expect(list[0].Line).To(HaveLen(2))
			Expect(list[0].Line[0].Start).To(Equal(new(int64(16000))))
			Expect(list[0].Line[0].Value).To(Equal("First line"))
			Expect(list[0].Line[1].Start).To(Equal(new(int64(18000))))
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

			list, err := parseTTML("xxx", content)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].Line).To(HaveLen(2))
			Expect(list[0].Line[0].Start).To(Equal(new(int64(10170))))
			Expect(list[0].Line[0].Value).To(Equal("First line"))
			Expect(list[0].Line[1].Start).To(Equal(new(int64(13710))))
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

			list, err := parseTTML("xxx", content)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].Agents).To(Equal([]Agent{
				{ID: "main", Role: "main"},
				{ID: "__nd_bg__|main", Role: "bg"},
			}))
			Expect(list[0].Line).To(HaveLen(1))

			line := list[0].Line[0]
			Expect(line.Start).To(Equal(new(int64(1000))))
			Expect(line.Value).To(Equal("Hello echo"))
			Expect(line.End).To(Equal(new(int64(3000))))
			Expect(line.Cue).To(HaveLen(3))

			Expect(line.Cue[0]).To(Equal(Cue{Start: new(int64(1000)), End: new(int64(1400)), Value: "He", ByteStart: 0, ByteEnd: 1, AgentID: "main"}))
			Expect(line.Cue[1]).To(Equal(Cue{Start: new(int64(1400)), End: new(int64(1800)), Value: "llo", ByteStart: 2, ByteEnd: 4, AgentID: "main"}))
			Expect(line.Cue[2]).To(Equal(Cue{Start: new(int64(2000)), End: new(int64(2500)), Value: "echo", ByteStart: 6, ByteEnd: 9, AgentID: "__nd_bg__|main"}))
		})

		It("should append role tokens exactly instead of using substring matches", func() {
			content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml" xmlns:ttm="http://www.w3.org/ns/ttml#metadata">
  <body xml:lang="eng">
    <div>
      <p begin="00:01.000" end="00:03.000" ttm:role="not-x-bg"><span begin="00:01.000" end="00:01.400">Lead</span><span ttm:role="x-bg"><span begin="00:02.000" end="00:02.500">Echo</span></span></p>
    </div>
  </body>
</tt>`)

			list, err := parseTTML("xxx", content)

			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].Agents).To(Equal([]Agent{
				{ID: "main", Role: "main"},
				{ID: "__nd_bg__|main", Role: "bg"},
			}))
			Expect(list[0].Line).To(HaveLen(1))
			Expect(list[0].Line[0].Cue).To(HaveLen(2))
			Expect(list[0].Line[0].Cue[0].AgentID).To(Equal("main"))
			Expect(list[0].Line[0].Cue[1].AgentID).To(Equal("__nd_bg__|main"))
		})

		It("should parse named TTML agents into main, voice, and group roles", func() {
			content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml" xmlns:ttm="http://www.w3.org/ns/ttml#metadata">
  <head>
    <metadata>
      <ttm:agent xml:id="v1" type="person"><ttm:name>Chris Martin</ttm:name></ttm:agent>
      <ttm:agent xml:id="v2" type="person"><ttm:name>Jin</ttm:name></ttm:agent>
      <ttm:agent xml:id="v1000" type="group"><ttm:name>All</ttm:name></ttm:agent>
    </metadata>
  </head>
  <body xml:lang="eng">
    <div>
      <p begin="1s" end="2s" ttm:agent="v1"><span begin="1s" end="1.5s">You</span></p>
      <p begin="2s" end="3s" ttm:agent="v2"><span begin="2s" end="2.5s">and</span></p>
      <p begin="3s" end="4s" ttm:agent="v1000"><span begin="3s" end="3.5s">All</span></p>
    </div>
  </body>
</tt>`)

			list, err := parseTTML("xxx", content)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].Agents).To(Equal([]Agent{
				{ID: "v1", Role: "main", Name: "Chris Martin"},
				{ID: "v2", Role: "voice", Name: "Jin"},
				{ID: "v1000", Role: "group", Name: "All"},
			}))
			Expect(list[0].Line[0].Cue[0].AgentID).To(Equal("v1"))
			Expect(list[0].Line[1].Cue[0].AgentID).To(Equal("v2"))
			Expect(list[0].Line[2].Cue[0].AgentID).To(Equal("v1000"))
		})

		It("should avoid collisions between derived background agents and explicit TTML agent ids", func() {
			content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml" xmlns:ttm="http://www.w3.org/ns/ttml#metadata">
  <head>
    <metadata>
      <ttm:agent xml:id="lead" type="person"><ttm:name>Lead</ttm:name></ttm:agent>
      <ttm:agent xml:id="lead__bg" type="person"><ttm:name>Existing Background Id</ttm:name></ttm:agent>
    </metadata>
  </head>
  <body xml:lang="eng">
    <div>
      <p begin="1s" end="2s" ttm:agent="lead">
        <span begin="1s" end="1.4s">Lead</span>
        <span ttm:role="x-bg"><span begin="1.5s" end="1.8s">Echo</span></span>
      </p>
      <p begin="2s" end="3s" ttm:agent="lead__bg">
        <span begin="2s" end="2.5s">Named</span>
      </p>
    </div>
  </body>
</tt>`)

			list, err := parseTTML("xxx", content)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].Agents).To(Equal([]Agent{
				{ID: "lead", Role: "main", Name: "Lead"},
				{ID: "__nd_bg__|lead", Role: "bg", Name: "Lead"},
				{ID: "lead__bg", Role: "voice", Name: "Existing Background Id"},
			}))
			Expect(list[0].Line).To(HaveLen(2))
			Expect(list[0].Line[0].Cue).To(HaveLen(2))
			Expect(list[0].Line[0].Cue[0].AgentID).To(Equal("lead"))
			Expect(list[0].Line[0].Cue[1].AgentID).To(Equal("__nd_bg__|lead"))
			Expect(list[0].Line[1].Cue).To(HaveLen(1))
			Expect(list[0].Line[1].Cue[0].AgentID).To(Equal("lead__bg"))
		})

		It("should fill missing cue agent ids with the resolved main agent", func() {
			content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml" xmlns:ttm="http://www.w3.org/ns/ttml#metadata">
  <head>
    <metadata>
      <ttm:agent xml:id="guest" type="person"><ttm:name>Guest Vocal</ttm:name></ttm:agent>
    </metadata>
  </head>
  <body xml:lang="eng">
    <div>
      <p begin="1s" end="3s">
        <span begin="1s" end="1.4s">Lead</span>
        <span begin="2s" end="2.4s" ttm:agent="guest">Guest</span>
      </p>
    </div>
  </body>
</tt>`)

			list, err := parseTTML("xxx", content)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].Agents).To(Equal([]Agent{
				{ID: "guest", Role: "main", Name: "Guest Vocal"},
			}))
			Expect(list[0].Line).To(HaveLen(1))
			Expect(list[0].Line[0].Cue).To(HaveLen(2))
			Expect(list[0].Line[0].Cue[0].AgentID).To(Equal("guest"))
			Expect(list[0].Line[0].Cue[1].AgentID).To(Equal("guest"))
		})
	})

	Describe("Whitespace handling", func() {
		It("should collapse pretty-print indentation between spans into single spaces, not line breaks", func() {
			content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml" xmlns:ttm="http://www.w3.org/ns/ttml#metadata">
  <body xml:lang="eng">
    <div>
      <p begin="1:22.889" end="1:26.859" ttm:agent="v2">
        <span begin="1:22.889" end="1:23.127">It</span>
        <span begin="1:23.374" end="1:23.938">in,</span>
        <span ttm:role="x-bg">
          <span begin="1:23.881" end="1:24.243">(When you</span>
          <span begin="1:26.232" end="1:26.859">slide)</span>
        </span>
      </p>
    </div>
  </body>
</tt>`)

			list, err := parseTTML("xxx", content)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].Line).To(HaveLen(1))

			line := list[0].Line[0]
			Expect(line.Value).To(Equal("It in, (When you slide)"))
			Expect(line.Value).ToNot(ContainSubstring("\n"))
			Expect(line.Cue).To(HaveLen(4))
			Expect(line.Cue[0]).To(Equal(Cue{Start: new(int64(82889)), End: new(int64(83127)), Value: "It", ByteStart: 0, ByteEnd: 1, AgentID: "v2"}))
			Expect(line.Cue[1]).To(Equal(Cue{Start: new(int64(83374)), End: new(int64(83938)), Value: "in,", ByteStart: 3, ByteEnd: 5, AgentID: "v2"}))
			Expect(line.Cue[2]).To(Equal(Cue{Start: new(int64(83881)), End: new(int64(84243)), Value: "(When you", ByteStart: 7, ByteEnd: 15, AgentID: "__nd_bg__|v2"}))
			Expect(line.Cue[3]).To(Equal(Cue{Start: new(int64(86232)), End: new(int64(86859)), Value: "slide)", ByteStart: 17, ByteEnd: 22, AgentID: "__nd_bg__|v2"}))
		})

		It("should preserve explicit <br/> as a line break", func() {
			content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml" xmlns:ttm="http://www.w3.org/ns/ttml#metadata">
  <body xml:lang="eng">
    <div>
      <p begin="00:01.000" end="00:03.000">
        <span begin="00:01.000" end="00:01.400">first</span>
        <br/>
        <span begin="00:02.000" end="00:02.500">second</span>
      </p>
    </div>
  </body>
</tt>`)

			list, err := parseTTML("xxx", content)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].Line).To(HaveLen(1))
			Expect(list[0].Line[0].Value).To(Equal("first\nsecond"))
		})

		It("should only collapse XML whitespace, leaving other Unicode spaces intact", func() {
			// Whitespace collapsing only touches the XML S characters
			// (space/tab/CR/LF). Other Unicode spaces like U+3000 are left as-is:
			// the U+3000 inside a span survives, while the pretty-print newline
			// between spans still collapses to a single space.
			content := []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" +
				"<tt xmlns=\"http://www.w3.org/ns/ttml\">\n" +
				"  <body xml:lang=\"jpn\">\n" +
				"    <div>\n" +
				"      <p begin=\"00:01.000\" end=\"00:03.000\">\n" +
				"        <span begin=\"00:01.000\" end=\"00:01.400\">あ　い</span>\n" +
				"        <span begin=\"00:02.000\" end=\"00:02.500\">う</span>\n" +
				"      </p>\n" +
				"    </div>\n" +
				"  </body>\n" +
				"</tt>")

			list, err := parseTTML("xxx", content)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].Line).To(HaveLen(1))
			Expect(list[0].Line[0].Value).To(Equal("あ　い う"))
			Expect(list[0].Line[0].Cue[0].Value).To(Equal("あ　い"))
		})
	})

	Describe("Interleaved background cue timing", func() {
		It("should not corrupt a main cue's end time when a background cue is earlier in time", func() {
			// Background spans (x-bg) appear after the main spans in document order
			// but their timings interleave with the main timeline. End-time
			// normalization must be per agent so the last main cue keeps its real
			// end instead of collapsing to its own start.
			content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tt xmlns="http://www.w3.org/ns/ttml" xmlns:ttm="http://www.w3.org/ns/ttml#metadata">
  <body xml:lang="eng">
    <div>
      <p begin="1:22.889" end="1:26.859" ttm:agent="v2"><span begin="1:25.593" end="1:25.934">real</span> <span begin="1:25.934" end="1:26.751">slow</span> <span ttm:role="x-bg"><span begin="1:23.881" end="1:24.243">(When you</span> <span begin="1:26.232" end="1:26.859">slide)</span></span></p>
    </div>
  </body>
</tt>`)

			list, err := parseTTML("xxx", content)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].Line).To(HaveLen(1))

			line := list[0].Line[0]
			Expect(line.Cue).To(HaveLen(4))

			cuesByAgent := map[string][]Cue{}
			for _, c := range line.Cue {
				cuesByAgent[c.AgentID] = append(cuesByAgent[c.AgentID], c)
			}

			mainCues := cuesByAgent["v2"]
			Expect(mainCues).To(HaveLen(2))
			Expect(*mainCues[0].End).To(Equal(int64(85934))) // "real"
			// "slow" must keep its real end (86751), not collapse to its start.
			Expect(*mainCues[1].Start).To(Equal(int64(85934)))
			Expect(*mainCues[1].End).To(Equal(int64(86751)))

			bgCues := cuesByAgent["__nd_bg__|v2"]
			Expect(bgCues).To(HaveLen(2))
			Expect(*bgCues[0].End).To(Equal(int64(84243))) // "(When you"
			Expect(*bgCues[1].End).To(Equal(int64(86859))) // "slide)"
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

			list, err := parseTTML("xxx", content)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].Line).To(HaveLen(1))

			line := list[0].Line[0]
			Expect(line.Start).To(Equal(new(int64(43444))))
			Expect(line.Value).To(Equal("go go"))
			Expect(line.End).To(Equal(new(int64(45570))))
			Expect(line.Cue).To(HaveLen(2))
			Expect(line.Cue[0]).To(Equal(Cue{Start: new(int64(43444)), End: new(int64(43716)), Value: "go", ByteStart: 0, ByteEnd: 1}))
			Expect(line.Cue[1]).To(Equal(Cue{Start: new(int64(43716)), End: new(int64(43887)), Value: "go", ByteStart: 3, ByteEnd: 4}))
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

			list, err := parseTTML("xxx", content)
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

			list, err := parseTTML("xxx", content)
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
			Expect(translation.Line[0].Start).To(Equal(new(int64(1000))))
			Expect(translation.Line[0].Value).To(Equal("Hola"))
			Expect(translation.Line[0].End).To(Equal(new(int64(1500))))

			By("checking the pronunciation track")
			pronunciation := list[2]
			Expect(pronunciation.Kind).To(Equal("pronunciation"))
			Expect(pronunciation.Lang).To(Equal("ja-latn"))
			Expect(pronunciation.Line).To(HaveLen(1))
			Expect(pronunciation.Line[0].Start).To(Equal(new(int64(2000))))
			Expect(pronunciation.Line[0].Value).To(Equal("konni"))
			Expect(pronunciation.Line[0].End).To(Equal(new(int64(2600))))
			Expect(pronunciation.Line[0].Cue).To(HaveLen(2))
			Expect(pronunciation.Line[0].Cue[0]).To(Equal(Cue{Start: new(int64(2000)), End: new(int64(2300)), Value: "ko", ByteStart: 0, ByteEnd: 1}))
			Expect(pronunciation.Line[0].Cue[1]).To(Equal(Cue{Start: new(int64(2300)), End: new(int64(2600)), Value: "nni", ByteStart: 2, ByteEnd: 4}))
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

			list, err := parseTTML("xxx", content)
			Expect(err).ToNot(HaveOccurred())

			var pronunciation *Lyrics
			for i := range list {
				if list[i].Kind == "pronunciation" {
					pronunciation = &list[i]
					break
				}
			}
			Expect(pronunciation).ToNot(BeNil())
			Expect(pronunciation.Line).To(HaveLen(1))

			line := pronunciation.Line[0]
			Expect(line.Start).To(Equal(new(int64(2747))))
			Expect(line.Value).To(Equal("I woke up"))
			Expect(line.Cue).To(HaveLen(3))
			Expect(line.Cue[0]).To(Equal(Cue{Start: new(int64(2747)), End: new(int64(3018)), Value: "I", ByteStart: 0, ByteEnd: 0}))
			Expect(line.Cue[1]).To(Equal(Cue{Start: new(int64(3018)), End: new(int64(3179)), Value: "woke", ByteStart: 2, ByteEnd: 5}))
			Expect(line.Cue[2]).To(Equal(Cue{Start: new(int64(3179)), End: new(int64(3582)), Value: "up", ByteStart: 7, ByteEnd: 8}))
		})
	})
})
