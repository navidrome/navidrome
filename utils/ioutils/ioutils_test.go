package ioutils

import (
	"bytes"
	"io"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/text/transform"
)

func TestIOUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "IO Utils Suite")
}

var _ = Describe("UTF8Reader", func() {
	Context("when reading text with UTF-8 BOM", func() {
		It("strips the UTF-8 BOM marker", func() {
			// UTF-8 BOM is EF BB BF
			input := []byte{0xEF, 0xBB, 0xBF, 'h', 'e', 'l', 'l', 'o'}
			reader := UTF8Reader(bytes.NewReader(input))

			output, err := io.ReadAll(reader)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(output)).To(Equal("hello"))
		})

		It("strips UTF-8 BOM from multi-line text", func() {
			// Test with the actual LRC file format
			input := []byte{0xEF, 0xBB, 0xBF, '[', '0', '0', ':', '0', '0', '.', '0', '0', ']', ' ', 't', 'e', 's', 't'}
			reader := UTF8Reader(bytes.NewReader(input))

			output, err := io.ReadAll(reader)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(output)).To(Equal("[00:00.00] test"))
		})
	})

	Context("when reading text without BOM", func() {
		It("passes through unchanged", func() {
			input := []byte("hello world")
			reader := UTF8Reader(bytes.NewReader(input))

			output, err := io.ReadAll(reader)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(output)).To(Equal("hello world"))
		})
	})

	Context("when reading UTF-16 LE encoded text", func() {
		It("converts to UTF-8 and strips BOM", func() {
			// UTF-16 LE BOM (FF FE) followed by "hi" in UTF-16 LE
			input := []byte{0xFF, 0xFE, 'h', 0x00, 'i', 0x00}
			reader := UTF8Reader(bytes.NewReader(input))

			output, err := io.ReadAll(reader)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(output)).To(Equal("hi"))
		})
	})

	Context("when reading UTF-16 BE encoded text", func() {
		It("converts to UTF-8 and strips BOM", func() {
			// UTF-16 BE BOM (FE FF) followed by "hi" in UTF-16 BE
			input := []byte{0xFE, 0xFF, 0x00, 'h', 0x00, 'i'}
			reader := UTF8Reader(bytes.NewReader(input))

			output, err := io.ReadAll(reader)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(output)).To(Equal("hi"))
		})
	})

	Context("when reading empty content", func() {
		It("returns empty string", func() {
			reader := UTF8Reader(bytes.NewReader([]byte{}))

			output, err := io.ReadAll(reader)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(output)).To(Equal(""))
		})
	})

	Context("when reading Windows-1252/Latin-1 encoded text (issue #6037)", func() {
		It("decodes accented bytes instead of emitting U+FFFD", func() {
			// "PokéMon" and "Và" with é (0xE9) and à (0xE0) as single Latin-1 bytes.
			input := []byte{'P', 'o', 'k', 0xE9, 'M', 'o', 'n', ' ', 'V', 0xE0}
			reader := UTF8Reader(bytes.NewReader(input))

			output, err := io.ReadAll(reader)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(output)).To(Equal("PokéMon Và"))
		})

		It("decodes Windows-1252 specific bytes in the 0x80-0x9F range", func() {
			// 0x80 is the Euro sign and 0x93/0x94 are curly quotes in Windows-1252,
			// none of which exist in plain Latin-1.
			input := []byte{0x80, '5', ' ', 0x93, 'h', 'i', 0x94}
			reader := UTF8Reader(bytes.NewReader(input))

			output, err := io.ReadAll(reader)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(output)).To(Equal("€5 “hi”"))
		})

		It("leaves valid multi-byte UTF-8 untouched", func() {
			// A path that mixes CJK and accented Latin, already valid UTF-8, must
			// pass through byte-for-byte so real UTF-8 files are never corrupted.
			input := []byte("收藏/PokéMon Và White.mp3")
			reader := UTF8Reader(bytes.NewReader(input))

			output, err := io.ReadAll(reader)
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(Equal(input))
		})

		It("preserves a genuine U+FFFD present in valid UTF-8", func() {
			input := []byte("a�b")
			reader := UTF8Reader(bytes.NewReader(input))

			output, err := io.ReadAll(reader)
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(Equal(input))
		})
	})
})

var _ = Describe("utf8OrWindows1252 transformer", func() {
	var t utf8OrWindows1252

	It("has a no-op Reset", func() {
		Expect(t.Reset).ToNot(Panic())
	})

	It("returns ErrShortDst when an ASCII byte does not fit the destination", func() {
		nDst, nSrc, err := t.Transform(make([]byte, 0), []byte("a"), true)

		Expect(err).To(MatchError(transform.ErrShortDst))
		Expect(nDst).To(Equal(0))
		Expect(nSrc).To(Equal(0))
	})

	It("returns ErrShortDst when a valid multi-byte UTF-8 rune does not fit", func() {
		// "é" is 0xC3 0xA9 in UTF-8 and needs a 2-byte destination.
		nDst, nSrc, err := t.Transform(make([]byte, 1), []byte("é"), true)

		Expect(err).To(MatchError(transform.ErrShortDst))
		Expect(nDst).To(Equal(0))
		Expect(nSrc).To(Equal(0))
	})

	It("returns ErrShortDst when a Windows-1252 decoded rune does not fit", func() {
		// 0xE9 decodes to "é", whose UTF-8 form needs 2 bytes.
		nDst, nSrc, err := t.Transform(make([]byte, 1), []byte{0xE9}, true)

		Expect(err).To(MatchError(transform.ErrShortDst))
		Expect(nDst).To(Equal(0))
		Expect(nSrc).To(Equal(0))
	})

	It("returns ErrShortSrc for an incomplete trailing sequence when more may follow", func() {
		// 0xC3 starts a 2-byte sequence; with no second byte and not at EOF, the
		// decoder must wait for more input rather than decode it as Latin-1.
		nDst, nSrc, err := t.Transform(make([]byte, 8), []byte{0xC3}, false)

		Expect(err).To(MatchError(transform.ErrShortSrc))
		Expect(nDst).To(Equal(0))
		Expect(nSrc).To(Equal(0))
	})

	It("maps an undefined Windows-1252 byte to U+FFFD at EOF", func() {
		// 0x81 is undefined in Windows-1252, so it decodes to the replacement char.
		dst := make([]byte, 8)
		nDst, nSrc, err := t.Transform(dst, []byte{0x81}, true)

		Expect(err).ToNot(HaveOccurred())
		Expect(nSrc).To(Equal(1))
		Expect(string(dst[:nDst])).To(Equal("�"))
	})
})

var _ = Describe("UTF8ReadFile", func() {
	Context("when reading a file with UTF-8 BOM", func() {
		It("strips the BOM marker", func() {
			// Use the actual fixture from issue #4631
			contents, err := UTF8ReadFile("../../tests/fixtures/bom-test.lrc")
			Expect(err).ToNot(HaveOccurred())

			// Should NOT start with BOM
			Expect(contents[0]).ToNot(Equal(byte(0xEF)))
			// Should start with '['
			Expect(contents[0]).To(Equal(byte('[')))
			Expect(string(contents)).To(HavePrefix("[00:00.00]"))
		})
	})

	Context("when reading a file without BOM", func() {
		It("reads the file normally", func() {
			contents, err := UTF8ReadFile("../../tests/fixtures/test.lrc")
			Expect(err).ToNot(HaveOccurred())

			// Should contain the expected content
			Expect(string(contents)).To(ContainSubstring("We're no strangers to love"))
		})
	})

	Context("when reading a non-existent file", func() {
		It("returns an error", func() {
			_, err := UTF8ReadFile("../../tests/fixtures/nonexistent.lrc")
			Expect(err).To(HaveOccurred())
		})
	})
})
