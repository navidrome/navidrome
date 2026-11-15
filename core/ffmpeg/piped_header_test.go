package ffmpeg

import (
	"bytes"
	"errors"
	"io"
	"testing/iotest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func readPatched(in []byte, duration float32) []byte {
	out, err := io.ReadAll(patchPipedHeader(io.NopCloser(bytes.NewReader(in)), duration))
	Expect(err).ToNot(HaveOccurred())
	return out
}

var _ = Describe("patchPipedHeader", func() {
	It("passes through when the duration is zero or negative", func() {
		in := monoMP3()
		Expect(readPatched(in, 0)).To(Equal(in))
		Expect(readPatched(in, -5)).To(Equal(in))
	})

	It("passes through a stream that is neither FLAC nor mp3", func() {
		in := []byte("OggS\x00\x02\x00\x00\x00\x00\x00\x00\x00\x00 and then some bytes")
		Expect(readPatched(in, 1.0)).To(Equal(in))
	})

	It("passes through an ADTS stream, whose sync word looks like mp3's", func() {
		in := append([]byte{0xFF, 0xF1, 0x50, 0x80, 0x4D, 0x1F, 0xFC}, make([]byte, 600)...)
		Expect(readPatched(in, 1.0)).To(Equal(in))
	})

	It("passes through an empty stream", func() {
		Expect(readPatched(nil, 1.0)).To(BeEmpty())
	})

	It("propagates a read error from the underlying stream", func() {
		r := patchPipedHeader(io.NopCloser(iotest.ErrReader(errors.New("ffmpeg died"))), 1.0)
		_, err := io.ReadAll(r)
		Expect(err).To(MatchError(ContainSubstring("ffmpeg died")))
	})

	It("closes the underlying stream", func() {
		c := &closeSpy{Reader: bytes.NewReader(monoMP3())}
		Expect(patchPipedHeader(c, 1.0).Close()).To(Succeed())
		Expect(c.closed).To(BeTrue())
	})
})

type closeSpy struct {
	io.Reader
	closed bool
}

func (c *closeSpy) Close() error { c.closed = true; return nil }
