package model_test

import (
	. "github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParseLyricsfile", func() {
	It("rebuilds CueLine.Value from word cues so byteStart/byteEnd remain valid offsets", func() {
		yaml := `version: '1.0'
metadata:
  title: 'Punctuated'
  artist: 'Test Artist'
lines:
  - text: 'Hello, world!'
    start_ms: 1000
    end_ms: 3000
    words:
      - text: 'Hello '
        start_ms: 1000
        end_ms: 2000
      - text: 'world'
        start_ms: 2000
        end_ms: 3000
`
		lyrics, err := ParseLyricsfile(yaml)
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.CueLine).To(HaveLen(1))
		cl := lyrics.CueLine[0]

		Expect(cl.Cue).To(HaveLen(2))
		Expect(cl.Cue[0].Value).To(Equal("Hello "))
		Expect(cl.Cue[1].Value).To(Equal("world"))
		Expect(cl.Value).To(Equal("Hello world"))
	})

	It("keeps entry.Text on CueLine.Value when the line has no word cues", func() {
		yaml := `version: '1.0'
metadata:
  title: 'Line only'
  artist: 'Test Artist'
lines:
  - text: 'Hello, world!'
    start_ms: 1000
    end_ms: 3000
`
		lyrics, err := ParseLyricsfile(yaml)
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.CueLine).To(HaveLen(1))
		cl := lyrics.CueLine[0]

		Expect(cl.Cue).To(BeEmpty())
		Expect(cl.Value).To(Equal("Hello, world!"))
	})
})
