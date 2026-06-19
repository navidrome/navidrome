package model_test

import (
	. "github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParseLyricsfile", func() {
	DescribeTable("returns nil,nil for YAML without the Lyricsfile version marker",
		func(input string) {
			lyrics, err := ParseLyricsfile(input)
			Expect(err).ToNot(HaveOccurred())
			Expect(lyrics).To(BeNil())
		},
		Entry("arbitrary YAML", "hello: world\n"),
		Entry("Lyricsfile-shaped but unversioned", `metadata:
  title: 'Looks close'
lines:
  - text: "But should not be claimed"
    start_ms: 1000
`),
	)

	It("returns an error for invalid YAML", func() {
		_, err := ParseLyricsfile("not: valid: yaml: [")
		Expect(err).To(HaveOccurred())
	})

	It("parses line-level metadata without cues", func() {
		input := `version: '1.0'
metadata:
  title: 'Sample Track'
  artist: 'Test Artist'
  language: 'eng'
  offset_ms: -100
lines:
  - text: "We're no strangers to love"
    start_ms: 18800
  - text: "You know the rules and so do I"
    start_ms: 22801
`
		lyrics, err := ParseLyricsfile(input)
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics).To(HaveLen(1))

		l := lyrics[0]
		Expect(l.Kind).To(Equal("main"))
		Expect(l.Lang).To(Equal("eng"))
		Expect(l.DisplayArtist).To(Equal("Test Artist"))
		Expect(l.DisplayTitle).To(Equal("Sample Track"))
		Expect(l.Synced).To(BeTrue())
		Expect(l.Offset).ToNot(BeNil())
		Expect(*l.Offset).To(Equal(int64(-100)))
		Expect(l.Agents).To(BeNil())

		Expect(l.Line).To(HaveLen(2))
		Expect(*l.Line[0].Start).To(Equal(int64(18800)))
		Expect(l.Line[0].End).ToNot(BeNil())
		Expect(*l.Line[0].End).To(Equal(int64(22801)))
		Expect(l.Line[0].Value).To(Equal("We're no strangers to love"))
		Expect(l.Line[0].Cue).To(BeNil())

		Expect(*l.Line[1].Start).To(Equal(int64(22801)))
		Expect(l.Line[1].End).To(BeNil())
		Expect(l.Line[1].Value).To(Equal("You know the rules and so do I"))
		Expect(l.Line[1].Cue).To(BeNil())
	})

	It("parses plain-only Lyricsfile lyrics as unsynced lines", func() {
		input := `version: '1.0'
metadata:
  title: 'Plain Track'
  artist: 'Plain Artist'
  language: 'en'
lines: []
plain: |
  [Verse 1]
  First line

  Second line
`
		lyrics, err := ParseLyricsfile(input)
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics).To(HaveLen(1))

		l := lyrics[0]
		Expect(l.Kind).To(Equal("main"))
		Expect(l.Lang).To(Equal("en"))
		Expect(l.DisplayArtist).To(Equal("Plain Artist"))
		Expect(l.DisplayTitle).To(Equal("Plain Track"))
		Expect(l.Synced).To(BeFalse())
		Expect(l.Agents).To(BeNil())
		Expect(l.Line).To(Equal([]Line{
			{Value: "[Verse 1]"},
			{Value: "First line"},
			{Value: "Second line"},
		}))
	})

	It("produces word cues with inclusive UTF-8 byte offsets for monophonic word data", func() {
		input := `version: '1.0'
metadata:
  title: 'Karaoke'
  artist: 'Singer'
  language: 'eng'
lines:
  - text: "Hello world"
    start_ms: 1000
    end_ms: 3000
    words:
      - text: "Hello "
        start_ms: 1000
        end_ms: 1500
      - text: "world"
        start_ms: 1500
        end_ms: 3000
`
		lyrics, err := ParseLyricsfile(input)
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics).To(HaveLen(1))

		l := lyrics[0]
		Expect(l.Synced).To(BeTrue())
		Expect(l.Agents).To(BeNil())
		Expect(l.Line).To(HaveLen(1))

		line := l.Line[0]
		Expect(*line.Start).To(Equal(int64(1000)))
		Expect(*line.End).To(Equal(int64(3000)))
		Expect(line.Value).To(Equal("Hello world"))
		Expect(line.Cue).To(HaveLen(2))

		Expect(*line.Cue[0].Start).To(Equal(int64(1000)))
		Expect(*line.Cue[0].End).To(Equal(int64(1500)))
		Expect(line.Cue[0].Value).To(Equal("Hello "))
		Expect(line.Cue[0].ByteStart).To(Equal(0))
		Expect(line.Cue[0].ByteEnd).To(Equal(5))
		Expect(line.Cue[0].AgentID).To(Equal(""))

		Expect(*line.Cue[1].Start).To(Equal(int64(1500)))
		Expect(*line.Cue[1].End).To(Equal(int64(3000)))
		Expect(line.Cue[1].Value).To(Equal("world"))
		Expect(line.Cue[1].ByteStart).To(Equal(6))
		Expect(line.Cue[1].ByteEnd).To(Equal(10))
		Expect(line.Cue[1].AgentID).To(Equal(""))
	})

	It("prefers final word end_ms over next line start when inferring line end", func() {
		input := `version: '1.0'
metadata:
  title: 'Overlap From Words'
lines:
  - text: "Long vocal"
    start_ms: 1000
    words:
      - text: "Long "
        start_ms: 1000
        end_ms: 2000
      - text: "vocal"
        start_ms: 2000
        end_ms: 4000
  - text: "echo"
    start_ms: 3000
    end_ms: 3500
    words:
      - text: "echo"
        start_ms: 3000
        end_ms: 3500
`
		lyrics, err := ParseLyricsfile(input)
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics).To(HaveLen(1))

		l := lyrics[0]
		Expect(l.Agents).To(Equal([]Agent{
			{ID: "voice-0", Role: "main"},
			{ID: "voice-1", Role: "voice"},
		}))
		Expect(l.Line).To(HaveLen(2))
		Expect(l.Line[0].End).ToNot(BeNil())
		Expect(*l.Line[0].End).To(Equal(int64(4000)))
		Expect(l.Line[0].Cue[1].End).To(Equal(l.Line[0].End))
		Expect(l.Line[1].Cue[0].AgentID).To(Equal("voice-1"))
	})

	It("synthesises voice agents for overlapping lines and attributes per-cue", func() {
		input := `version: '1.0'
metadata:
  title: 'Duet'
lines:
  - text: "Lead vocal"
    start_ms: 1000
    end_ms: 4000
    words:
      - text: "Lead "
        start_ms: 1000
        end_ms: 2000
      - text: "vocal"
        start_ms: 2000
        end_ms: 4000
  - text: "echo"
    start_ms: 2000
    end_ms: 3000
    words:
      - text: "echo"
        start_ms: 2000
        end_ms: 3000
`
		lyrics, err := ParseLyricsfile(input)
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics).To(HaveLen(1))

		l := lyrics[0]
		Expect(l.Agents).To(Equal([]Agent{
			{ID: "voice-0", Role: "main"},
			{ID: "voice-1", Role: "voice"},
		}))
		Expect(l.Line).To(HaveLen(2))

		Expect(l.Line[0].Value).To(Equal("Lead vocal"))
		Expect(*l.Line[0].Start).To(Equal(int64(1000)))
		Expect(*l.Line[0].End).To(Equal(int64(4000)))
		Expect(l.Line[0].Cue).To(HaveLen(2))
		Expect(l.Line[0].Cue[0].AgentID).To(Equal("voice-0"))
		Expect(l.Line[0].Cue[1].AgentID).To(Equal("voice-0"))
		Expect(l.Line[0].Cue[0].ByteStart).To(Equal(0))
		Expect(l.Line[0].Cue[0].ByteEnd).To(Equal(4))
		Expect(l.Line[0].Cue[1].ByteStart).To(Equal(5))
		Expect(l.Line[0].Cue[1].ByteEnd).To(Equal(9))

		Expect(l.Line[1].Value).To(Equal("echo"))
		Expect(*l.Line[1].Start).To(Equal(int64(2000)))
		Expect(*l.Line[1].End).To(Equal(int64(3000)))
		Expect(l.Line[1].Cue).To(HaveLen(1))
		Expect(l.Line[1].Cue[0].AgentID).To(Equal("voice-1"))
		Expect(l.Line[1].Cue[0].ByteStart).To(Equal(0))
		Expect(l.Line[1].Cue[0].ByteEnd).To(Equal(3))
	})

	It("emits empty lines with Synced=false for instrumental tracks", func() {
		input := `version: '1.0'
metadata:
  title: 'Solo Piano'
  artist: 'Composer'
  language: 'eng'
  instrumental: true
`
		lyrics, err := ParseLyricsfile(input)
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics).To(HaveLen(1))

		l := lyrics[0]
		Expect(l.Kind).To(Equal("main"))
		Expect(l.Lang).To(Equal("eng"))
		Expect(l.DisplayArtist).To(Equal("Composer"))
		Expect(l.DisplayTitle).To(Equal("Solo Piano"))
		Expect(l.Synced).To(BeFalse())
		Expect(l.Line).To(BeEmpty())
		Expect(l.Agents).To(BeNil())
	})

	It("strips agent attribution when overlapping lines carry no cues", func() {
		input := `version: '1.0'
lines:
  - text: "Lead"
    start_ms: 1000
    end_ms: 4000
  - text: "echo"
    start_ms: 2000
    end_ms: 3000
`
		lyrics, err := ParseLyricsfile(input)
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics).To(HaveLen(1))

		l := lyrics[0]
		Expect(l.Line).To(HaveLen(2))
		Expect(l.Agents).To(BeNil())
		Expect(l.Line[0].Cue).To(BeNil())
		Expect(l.Line[1].Cue).To(BeNil())
	})
})
