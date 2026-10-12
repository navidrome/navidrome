package ioutils

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"testing/iotest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

		It("decodes a Windows-1252 line even when its high bytes also form valid UTF-8 elsewhere on the line", func() {
			// "café/Â£.mp3" in Windows-1252: é is 0xE9 and "Â£" is the pair C2 A3.
			// The lone 0xE9 makes the line invalid UTF-8, so the whole line is
			// decoded as Windows-1252 and the pair comes out as "Â£".
			input := []byte{'c', 'a', 'f', 0xE9, '/', 0xC2, 0xA3, '.', 'm', 'p', '3'}
			reader := UTF8Reader(bytes.NewReader(input))

			output, err := io.ReadAll(reader)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(output)).To(Equal("café/Â£.mp3"))
		})

		It("keeps valid UTF-8 lines intact when another line is Windows-1252", func() {
			// One legacy line (a lone 0xE9) must not turn the Cyrillic and Japanese
			// paths around it into mojibake.
			input := []byte("Ночные Снайперы/Рубеж/01.mp3\n")
			input = append(input, 'c', 'a', 'f', 0xE9, '.', 'm', 'p', '3', '\n')
			input = append(input, []byte("25時、ナイトコードで。/01.mp3\n")...)
			reader := UTF8Reader(bytes.NewReader(input))

			output, err := io.ReadAll(reader)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(output)).To(Equal("Ночные Снайперы/Рубеж/01.mp3\ncafé.mp3\n25時、ナイトコードで。/01.mp3\n"))
		})

		It("decodes every line of a Windows-1252 file", func() {
			input := []byte{'P', 'o', 'k', 0xE9, '\r', '\n', 'V', 0xE0, '\r', '\n'}
			reader := UTF8Reader(bytes.NewReader(input))

			output, err := io.ReadAll(reader)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(output)).To(Equal("Poké\r\nVà\r\n"))
		})

		It("reads a Windows-1252 line that has no trailing newline", func() {
			input := []byte("ok.mp3\ncaf")
			input = append(input, 0xE9)
			reader := UTF8Reader(bytes.NewReader(input))

			output, err := io.ReadAll(reader)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(output)).To(Equal("ok.mp3\ncafé"))
		})

		It("keeps a C2 A3 pair as £ when the whole file is valid UTF-8", func() {
			// The same byte pair, in a file that is valid UTF-8 throughout, is the
			// pound sign and must be left alone.
			input := []byte("cost/£.mp3")
			reader := UTF8Reader(bytes.NewReader(input))

			output, err := io.ReadAll(reader)
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(Equal(input))
		})
	})

	Context("when a fallback callback is given", func() {
		It("reports how many lines were decoded as Windows-1252", func() {
			input := []byte("fine.mp3\n")
			input = append(input, 'a', 0xE9, '\n', 'b', 0xE0, '\n')
			var got int
			reader := UTF8ReaderWithFallback(bytes.NewReader(input), func(lines int) { got = lines })

			_, err := io.ReadAll(reader)
			Expect(err).ToNot(HaveOccurred())
			Expect(got).To(Equal(2))
		})

		It("is not called for valid UTF-8", func() {
			called := false
			reader := UTF8ReaderWithFallback(bytes.NewReader([]byte("café.mp3\n")), func(int) { called = true })

			_, err := io.ReadAll(reader)
			Expect(err).ToNot(HaveOccurred())
			Expect(called).To(BeFalse())
		})

		It("is not called when the input starts with a BOM", func() {
			called := false
			input := []byte{0xEF, 0xBB, 0xBF, 'a', 0xE9}
			reader := UTF8ReaderWithFallback(bytes.NewReader(input), func(int) { called = true })

			_, err := io.ReadAll(reader)
			Expect(err).ToNot(HaveOccurred())
			Expect(called).To(BeFalse())
		})
	})

	Context("when the underlying reader fails", func() {
		It("surfaces the read error", func() {
			boom := errors.New("boom")
			reader := UTF8Reader(io.MultiReader(bytes.NewReader([]byte("abc")), iotest.ErrReader(boom)))

			_, err := io.ReadAll(reader)
			Expect(err).To(MatchError(boom))
		})
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
