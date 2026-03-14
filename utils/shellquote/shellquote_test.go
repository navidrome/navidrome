package shellquote_test

import (
	"testing"

	"github.com/navidrome/navidrome/utils/shellquote"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestShellquote(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Shellquote Suite")
}

var _ = Describe("Split", func() {
	It("splits simple space-separated words", func() {
		words, err := shellquote.Split("a b c")
		Expect(err).ToNot(HaveOccurred())
		Expect(words).To(Equal([]string{"a", "b", "c"}))
	})

	It("handles multiple spaces between words", func() {
		words, err := shellquote.Split("a    b")
		Expect(err).ToNot(HaveOccurred())
		Expect(words).To(Equal([]string{"a", "b"}))
	})

	It("handles single-quoted strings", func() {
		words, err := shellquote.Split("'hello world'")
		Expect(err).ToNot(HaveOccurred())
		Expect(words).To(Equal([]string{"hello world"}))
	})

	It("handles double-quoted strings", func() {
		words, err := shellquote.Split(`"hello world"`)
		Expect(err).ToNot(HaveOccurred())
		Expect(words).To(Equal([]string{"hello world"}))
	})

	It("handles backslash escapes in unquoted mode", func() {
		words, err := shellquote.Split(`hello\ world`)
		Expect(err).ToNot(HaveOccurred())
		Expect(words).To(Equal([]string{"hello world"}))
	})

	It("handles escaped quotes inside double quotes", func() {
		words, err := shellquote.Split(`"hello \" world"`)
		Expect(err).ToNot(HaveOccurred())
		Expect(words).To(Equal([]string{`hello " world`}))
	})

	It("handles mixed quoting in a single argument", func() {
		words, err := shellquote.Split("he'llo wo'rld")
		Expect(err).ToNot(HaveOccurred())
		Expect(words).To(Equal([]string{"hello world"}))
	})

	It("returns empty slice for empty input", func() {
		words, err := shellquote.Split("")
		Expect(err).ToNot(HaveOccurred())
		Expect(words).To(BeEmpty())
	})

	It("returns empty slice for whitespace-only input", func() {
		words, err := shellquote.Split("   ")
		Expect(err).ToNot(HaveOccurred())
		Expect(words).To(BeEmpty())
	})

	It("returns error for unterminated single quote", func() {
		_, err := shellquote.Split("'hello")
		Expect(err).To(MatchError(shellquote.ErrUnterminatedSingleQuote))
	})

	It("returns error for unterminated double quote", func() {
		_, err := shellquote.Split(`"hello`)
		Expect(err).To(MatchError(shellquote.ErrUnterminatedDoubleQuote))
	})

	It("returns error for unterminated escape", func() {
		_, err := shellquote.Split(`hello\`)
		Expect(err).To(MatchError(shellquote.ErrUnterminatedEscape))
	})

	It("handles tabs and newlines as delimiters", func() {
		words, err := shellquote.Split("a\tb\nc")
		Expect(err).ToNot(HaveOccurred())
		Expect(words).To(Equal([]string{"a", "b", "c"}))
	})

	It("parses the default MPV command template", func() {
		words, err := shellquote.Split("mpv --audio-device=%d --no-audio-display --pause %f --input-ipc-server=%s")
		Expect(err).ToNot(HaveOccurred())
		Expect(words).To(HaveLen(6))
		Expect(words).To(Equal([]string{
			"mpv",
			"--audio-device=%d",
			"--no-audio-display",
			"--pause",
			"%f",
			"--input-ipc-server=%s",
		}))
	})

	It("preserves spaces in quoted paths", func() {
		words, err := shellquote.Split(`--ao-pcm-file="/audio/my folder/file"`)
		Expect(err).ToNot(HaveOccurred())
		Expect(words).To(Equal([]string{`--ao-pcm-file=/audio/my folder/file`}))
	})

	It("handles backslash in double quotes for special chars", func() {
		words, err := shellquote.Split(`"hello\\world"`)
		Expect(err).ToNot(HaveOccurred())
		Expect(words).To(Equal([]string{`hello\world`}))
	})

	It("preserves backslash in double quotes for non-special chars", func() {
		words, err := shellquote.Split(`"hello\nworld"`)
		Expect(err).ToNot(HaveOccurred())
		Expect(words).To(Equal([]string{`hello\nworld`}))
	})

	It("handles escaped newline in double quotes", func() {
		words, err := shellquote.Split("\"hello\\\nworld\"")
		Expect(err).ToNot(HaveOccurred())
		Expect(words).To(Equal([]string{"helloworld"}))
	})

	// Cases from original go-shellquote test suite
	It("handles shell glob characters as literals", func() {
		words, err := shellquote.Split("glob* test?")
		Expect(err).ToNot(HaveOccurred())
		Expect(words).To(Equal([]string{"glob*", "test?"}))
	})

	It("handles backslash-escaped special characters", func() {
		words, err := shellquote.Split("don\\'t you know the dewey decimal system\\?")
		Expect(err).ToNot(HaveOccurred())
		Expect(words).To(Equal([]string{"don't", "you", "know", "the", "dewey", "decimal", "system?"}))
	})

	It("handles single-quote escape idiom", func() {
		// Shell idiom: end single-quote, escaped literal quote, start single-quote again
		words, err := shellquote.Split("'don'\\''t you know the dewey decimal system?'")
		Expect(err).ToNot(HaveOccurred())
		Expect(words).To(Equal([]string{"don't you know the dewey decimal system?"}))
	})

	It("handles empty string argument via quotes", func() {
		words, err := shellquote.Split("one '' two")
		Expect(err).ToNot(HaveOccurred())
		Expect(words).To(Equal([]string{"one", "", "two"}))
	})

	It("handles backslash-newline joining words in unquoted mode", func() {
		words, err := shellquote.Split("text with\\\na backslash-escaped newline")
		Expect(err).ToNot(HaveOccurred())
		Expect(words).To(Equal([]string{"text", "witha", "backslash-escaped", "newline"}))
	})

	It("handles quoted newline inside double quotes", func() {
		words, err := shellquote.Split("text \"with\na\" quoted newline")
		Expect(err).ToNot(HaveOccurred())
		Expect(words).To(Equal([]string{"text", "with\na", "quoted", "newline"}))
	})

	It("handles complex double-quoted escapes with backslash-newline", func() {
		words, err := shellquote.Split("\"quoted\\d\\\\\\\" text with\\\na backslash-escaped newline\"")
		Expect(err).ToNot(HaveOccurred())
		Expect(words).To(Equal([]string{"quoted\\d\\\" text witha backslash-escaped newline"}))
	})

	It("handles backslash-newline between words", func() {
		words, err := shellquote.Split("text with an escaped \\\n newline in the middle")
		Expect(err).ToNot(HaveOccurred())
		Expect(words).To(Equal([]string{"text", "with", "an", "escaped", "newline", "in", "the", "middle"}))
	})

	It("handles double-quoted substring concatenation", func() {
		words, err := shellquote.Split(`foo"bar"baz`)
		Expect(err).ToNot(HaveOccurred())
		Expect(words).To(Equal([]string{"foobarbaz"}))
	})

	It("returns error for unterminated quote after escape idiom", func() {
		_, err := shellquote.Split("'test'\\''ing")
		Expect(err).To(MatchError(shellquote.ErrUnterminatedSingleQuote))
	})

	It("returns error for unterminated double quote with single quote inside", func() {
		_, err := shellquote.Split("\"foo'bar")
		Expect(err).To(MatchError(shellquote.ErrUnterminatedDoubleQuote))
	})

	It("returns error for unterminated escape with leading whitespace", func() {
		_, err := shellquote.Split("   \\")
		Expect(err).To(MatchError(shellquote.ErrUnterminatedEscape))
	})
})
