package jsoncommentstrip_test

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/navidrome/navidrome/utils/jsoncommentstrip"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestJsonCommentStrip(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "JsonCommentStrip Suite")
}

var _ = Describe("NewReader", func() {
	read := func(input string) string {
		r := jsoncommentstrip.NewReader(strings.NewReader(input))
		out, err := io.ReadAll(r)
		Expect(err).ToNot(HaveOccurred())
		return string(out)
	}

	// compact returns the compacted JSON form of s, for readable comparisons.
	compact := func(s string) string {
		var buf bytes.Buffer
		ExpectWithOffset(1, json.Compact(&buf, []byte(s))).To(Succeed())
		return buf.String()
	}

	It("passes through JSON without comments unchanged", func() {
		input := `{"key": "value", "num": 42}`
		Expect(read(input)).To(Equal(input))
	})

	It("strips single-line comments", func() {
		input := `{
			// this is a comment
			"key": "value"
		}`
		Expect(compact(read(input))).To(Equal(compact(`{
			"key": "value"
		}`)))
	})

	It("strips single-line comments at end of line", func() {
		input := `{
			"key": "value" // inline comment
		}`
		Expect(compact(read(input))).To(Equal(compact(`{
			"key": "value"
		}`)))
	})

	It("strips block comments", func() {
		input := `{/* comment */"key": "value"}`
		Expect(compact(read(input))).To(Equal(`{"key":"value"}`))
	})

	It("strips multi-line block comments", func() {
		input := `{
			/* this is
			a multi-line
			comment */
			"key": "value"
		}`
		Expect(compact(read(input))).To(Equal(compact(`{
			"key": "value"
		}`)))
	})

	It("preserves // inside JSON strings", func() {
		input := `{"key": "value // not a comment"}`
		Expect(read(input)).To(Equal(input))
	})

	It("preserves /* inside JSON strings", func() {
		input := `{"key": "value /* not a comment */"}`
		Expect(read(input)).To(Equal(input))
	})

	It("handles escaped quotes in strings", func() {
		input := `{"key": "val\"ue // not a comment"}`
		Expect(read(input)).To(Equal(input))
	})

	It("handles / at end of input as literal", func() {
		input := `{"key": "value"}/`
		Expect(read(input)).To(Equal(input))
	})

	It("handles * inside block comment not followed by /", func() {
		input := `{/* a * b */"key": "value"}`
		Expect(compact(read(input))).To(Equal(`{"key":"value"}`))
	})

	It("handles empty input", func() {
		Expect(read("")).To(Equal(""))
	})

	It("handles mixed comments with real content", func() {
		input := `{
			// line comment
			"name": "test", /* inline block */
			/* multi
			   line */
			"value": "hello // world",
			"other": 123 // trailing
		}`
		Expect(compact(read(input))).To(Equal(compact(`{
			"name": "test",
			"value": "hello // world",
			"other": 123
		}`)))
	})

	It("handles consecutive slashes that are not comments", func() {
		input := `{"path": "/a/b"}`
		Expect(read(input)).To(Equal(input))
	})

	It("handles block comment at end of input", func() {
		input := `{"key": "value"}/* comment */`
		Expect(compact(read(input))).To(Equal(`{"key":"value"}`))
	})

	It("strips comment with windows-style line endings", func() {
		input := "{\r\n// comment\r\n\"key\": \"value\"\r\n}"
		Expect(compact(read(input))).To(Equal(compact(`{"key": "value"}`)))
	})

	It("strips line comments with mixed line endings", func() {
		// From original library: // comments with both \n and \r\n, including multiple on same line
		input := "{\n\"one\": 1, // test //\n\"two\": 2, //test //\r\n\"string\": \"value\"\n//test\n}"
		expected := "{\n\"one\": 1, \n\"two\": 2, \r\n\"string\": \"value\"\n\n}"
		Expect(read(input)).To(Equal(expected))
	})

	It("strips line comment at start of JSON", func() {
		// From original library: // comment as first thing in JSON
		input := "{// woot\n\"one\": 1, // test //\n\"two\": 2, //test //\r\n\"string\": \"value\"\n//test\n}"
		expected := "{\n\"one\": 1, \n\"two\": 2, \r\n\"string\": \"value\"\n\n}"
		Expect(read(input)).To(Equal(expected))
	})

	It("strips block comments with mixed line endings inside", func() {
		// From original library: block comment containing \r\n
		input := "{/* multi\nline\r\ncomment */\"one\":1}"
		expected := "{\"one\":1}"
		Expect(read(input)).To(Equal(expected))
	})

	It("handles complex mix of escaped quotes, comments, and strings", func() {
		// From original library TestQuotationEscape: escaped quote inside string followed by
		// comment-like chars, then real comments of both types
		input := "{/* multi\nline\r\ncomment */\"one\": \"a value \\\" // /*woot\"/* m\nl *///woot\r\n}"
		expected := "{\"one\": \"a value \\\" // /*woot\"\r\n}"
		Expect(read(input)).To(Equal(expected))
	})
})
