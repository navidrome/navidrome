package ffmpeg

import (
	"bytes"
	"errors"
	"io"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Decoded independently so the specs do not mirror the production bit-twiddling.
func readSampleRate(b []byte) int {
	return int(b[18])<<12 | int(b[19])<<4 | int(b[20])>>4
}

func readTotalSamples(b []byte) uint64 {
	return uint64(b[21]&0x0F)<<32 | uint64(b[22])<<24 | uint64(b[23])<<16 | uint64(b[24])<<8 | uint64(b[25])
}

var _ = Describe("patchFLACDuration", func() {
	var fileFLAC []byte

	// Zeroing total_samples reproduces what a piped transcode emits.
	pipedFLAC := func() []byte {
		b := bytes.Clone(fileFLAC)
		b[21] &= 0xF0
		clear(b[22:26])
		return b
	}

	readAll := func(in []byte, duration float32) []byte {
		out, err := io.ReadAll(patchFLACDuration(io.NopCloser(bytes.NewReader(in)), duration))
		Expect(err).ToNot(HaveOccurred())
		return out
	}

	BeforeEach(func() {
		var err error
		fileFLAC, err = os.ReadFile("tests/fixtures/test.flac")
		Expect(err).ToNot(HaveOccurred())
		Expect(readSampleRate(fileFLAC)).To(Equal(44100)) // specs below hard-code this rate
	})

	It("fills in total_samples from the duration", func() {
		out := readAll(pipedFLAC(), 1.0)
		Expect(readTotalSamples(out)).To(Equal(uint64(44100)))
	})

	It("takes the sample rate from the header, not from the source file", func() {
		in := pipedFLAC()
		// Rewrite the header's rate to 48000, as -ar would.
		in[18], in[19] = 0x0B, 0xB8
		in[20] &= 0x0F

		out := readAll(in, 2.0)

		Expect(readSampleRate(out)).To(Equal(48000))
		Expect(readTotalSamples(out)).To(Equal(uint64(96000)))
	})

	It("rounds to the nearest sample rather than truncating", func() {
		// float32(0.7)*44100 is 30869.9995, so truncation would lose a sample.
		out := readAll(pipedFLAC(), 0.7)
		Expect(readTotalSamples(out)).To(Equal(uint64(30870)))
	})

	It("passes through when the duration overflows the 36-bit field", func() {
		in := pipedFLAC()
		Expect(readAll(in, 2e6)).To(Equal(in))
	})

	It("leaves everything after the header untouched", func() {
		in := pipedFLAC()
		out := readAll(in, 1.0)
		Expect(out).To(HaveLen(len(in)))
		Expect(out[26:]).To(Equal(in[26:]))
		Expect(out[:18]).To(Equal(in[:18]))
	})

	It("leaves an already-populated total_samples alone", func() {
		out := readAll(fileFLAC, 99.0)
		Expect(out).To(Equal(fileFLAC))
	})

	It("passes through a stream that is not FLAC", func() {
		in := []byte("ID3\x04\x00\x00\x00\x00\x00\x00 not a flac stream at all, just bytes")
		Expect(readAll(in, 1.0)).To(Equal(in))
	})

	It("passes through when the first metadata block is not STREAMINFO", func() {
		in := pipedFLAC()
		in[4] = 0x04 // VORBIS_COMMENT
		Expect(readAll(in, 1.0)).To(Equal(in))
	})

	It("passes through a stream shorter than the STREAMINFO fields it patches", func() {
		in := pipedFLAC()[:20]
		Expect(readAll(in, 1.0)).To(Equal(in))
	})

	It("passes through an empty stream", func() {
		Expect(readAll(nil, 1.0)).To(BeEmpty())
	})

	It("passes through when the duration is zero or negative", func() {
		in := pipedFLAC()
		Expect(readAll(in, 0)).To(Equal(in))
		Expect(readAll(in, -5)).To(Equal(in))
	})

	It("passes through when the header declares no sample rate", func() {
		in := pipedFLAC()
		in[18], in[19] = 0, 0
		in[20] &= 0x0F
		Expect(readAll(in, 1.0)).To(Equal(in))
	})

	It("propagates a read error from the underlying stream", func() {
		_, err := io.ReadAll(patchFLACDuration(io.NopCloser(io.MultiReader(
			bytes.NewReader(pipedFLAC()[:10]), &errReader{})), 1.0))
		Expect(err).To(MatchError("boom"))
	})

	It("closes the underlying stream", func() {
		c := &closeSpy{Reader: bytes.NewReader(pipedFLAC())}
		Expect(patchFLACDuration(c, 1.0).Close()).To(Succeed())
		Expect(c.closed).To(BeTrue())
	})
})

type errReader struct{}

func (e *errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

type closeSpy struct {
	io.Reader
	closed bool
}

func (c *closeSpy) Close() error { c.closed = true; return nil }
