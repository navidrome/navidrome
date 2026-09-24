package ffmpeg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"testing/iotest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Offsets of the Xing tag inside the first frame, for the two layouts the specs use.
const (
	stereoTagOffset = 36    // 4 header + 32 side info
	monoTagOffset   = 21    // 4 header + 17 side info
	fixtureID3Len   = 39581 // where the first frame of tests/fixtures/test.mp3 starts
	fixtureFrameLen = 627   // its frame size, at 192kbps and 44100Hz
)

var _ = Describe("patchMP3Duration", func() {
	var pipedMP3 []byte

	// The fixture is an ffmpeg pipe's output in every way that matters here: ID3 tag,
	// frames, no Xing.
	BeforeEach(func() {
		var err error
		pipedMP3, err = os.ReadFile("tests/fixtures/test.mp3")
		Expect(err).ToNot(HaveOccurred())
		Expect(pipedMP3[fixtureID3Len : fixtureID3Len+2]).To(Equal([]byte{0xFF, 0xFB}))
		Expect(bytes.Contains(pipedMP3[:4096], []byte("Xing"))).To(BeFalse())
	})

	readAll := func(in []byte, duration float32) []byte {
		out, err := io.ReadAll(patchPipedHeader(io.NopCloser(bytes.NewReader(in)), duration))
		Expect(err).ToNot(HaveOccurred())
		return out
	}

	// Reads the tag and frame count of the frame starting at frameStart.
	readXing := func(b []byte, frameStart, tagOffset int) (string, uint32) {
		at := frameStart + tagOffset
		return string(b[at : at+4]), binary.BigEndian.Uint32(b[at+8:])
	}

	It("inserts a Xing frame declaring the duration in frames", func() {
		out := readAll(pipedMP3, 1.0)

		tag, frames := readXing(out, fixtureID3Len, stereoTagOffset)
		Expect(tag).To(Equal("Xing"))
		Expect(frames).To(Equal(uint32(38)), "1s at 44100Hz is 38 frames of 1152 samples")
		flags := binary.BigEndian.Uint32(out[fixtureID3Len+stereoTagOffset+4:])
		Expect(flags).To(Equal(uint32(1)), "only the frame count is present")
	})

	It("leaves the ID3 tag and the audio frames untouched", func() {
		out := readAll(pipedMP3, 1.0)

		start := fixtureID3Len
		Expect(out).To(HaveLen(len(pipedMP3) + fixtureFrameLen))
		Expect(out[:start]).To(Equal(pipedMP3[:start]))
		Expect(out[start+fixtureFrameLen:]).To(Equal(pipedMP3[start:]))
	})

	It("takes the sample rate from the header, not from the source file", func() {
		in := bytes.Clone(pipedMP3)
		start := fixtureID3Len
		in[start+2] = in[start+2]&^0x0C | 0x04 // sample rate index 1: 48000Hz

		out := readAll(in, 1.0)

		_, frames := readXing(out, start, stereoTagOffset)
		Expect(frames).To(Equal(uint32(42))) // 48000/1152, rounded
	})

	It("rounds the frame count to the nearest frame", func() {
		out := readAll(pipedMP3, 0.99) // 37.9 frames
		_, frames := readXing(out, fixtureID3Len, stereoTagOffset)
		Expect(frames).To(Equal(uint32(38)))
	})

	It("places the tag after the shorter side info of a mono stream", func() {
		in := monoMP3()

		out := readAll(in, 1.0)

		tag, frames := readXing(out, 0, monoTagOffset)
		Expect(tag).To(Equal("Xing"))
		Expect(frames).To(Equal(uint32(38)))
	})

	It("inserts the frame at the start of a stream with no ID3 tag", func() {
		in := pipedMP3[fixtureID3Len:]

		out := readAll(in, 1.0)

		tag, _ := readXing(out, 0, stereoTagOffset)
		Expect(tag).To(Equal("Xing"))
		Expect(out[fixtureFrameLen:]).To(Equal(in))
	})

	It("leaves a stream that already declares its duration alone", func() {
		patched := readAll(pipedMP3, 1.0)
		Expect(readAll(patched, 99.0)).To(Equal(patched))
	})

	It("passes through when the duration is zero or negative", func() {
		Expect(readAll(pipedMP3, 0)).To(Equal(pipedMP3))
		Expect(readAll(pipedMP3, -5)).To(Equal(pipedMP3))
	})

	It("passes through a stream that is not mp3", func() {
		in := []byte("fLaC\x00\x00\x00\x22 and then some bytes that are not frames")
		Expect(readAll(in, 1.0)).To(Equal(in))
	})

	It("passes through a frame header with a reserved bitrate or sample rate", func() {
		in := monoMP3()
		in[2] |= 0xF0 // bitrate index 15 is invalid
		Expect(readAll(in, 1.0)).To(Equal(in))

		in = monoMP3()
		in[2] |= 0x0C // sample rate index 3 is reserved
		Expect(readAll(in, 1.0)).To(Equal(in))
	})

	It("passes through a stream too short to hold a frame header", func() {
		in := monoMP3()[:3]
		Expect(readAll(in, 1.0)).To(Equal(in))
	})

	It("passes through a stream whose ID3 tag never ends", func() {
		in := bytes.Clone(pipedMP3[:200])
		Expect(readAll(in, 1.0)).To(Equal(in))
	})

	It("passes through an empty stream", func() {
		Expect(readAll(nil, 1.0)).To(BeEmpty())
	})

	It("propagates a read error from the underlying stream", func() {
		r := patchPipedHeader(io.NopCloser(iotest.ErrReader(errors.New("ffmpeg died"))), 1.0)
		_, err := io.ReadAll(r)
		Expect(err).To(MatchError(ContainSubstring("ffmpeg died")))
	})
})

// monoMP3 is one silent MPEG1 Layer III frame, 128kbps, 44100Hz, mono.
func monoMP3() []byte {
	b := make([]byte, 417)
	copy(b, []byte{0xFF, 0xFB, 0x90, 0xC0})
	return b
}
