package ioutils

import (
	"bytes"
	"io"
	"testing"

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
