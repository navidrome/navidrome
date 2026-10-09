package nativeapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("writeDeleteManyResponse", func() {
	var w *httptest.ResponseRecorder

	write := func(ids ...string) map[string]any {
		w = httptest.NewRecorder()
		writeDeleteManyResponse(w, httptest.NewRequest("DELETE", "/missing", nil), ids)

		var body map[string]any
		Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed(), "response body must be valid JSON: %s", w.Body.String())
		return body
	}

	It("returns a single id as an object", func() {
		Expect(write("abc123")).To(HaveKeyWithValue("id", "abc123"))
	})

	It("returns multiple ids as a list", func() {
		Expect(write("a", "b")).To(HaveKeyWithValue("ids", ConsistOf("a", "b")))
	})

	It("stays valid JSON when the id contains a backslash", func() {
		Expect(write(`a\`)).To(HaveKeyWithValue("id", `a\`))
	})

	It("stays valid JSON when the id contains a quote", func() {
		Expect(write(`a"b`)).To(HaveKeyWithValue("id", `a"b`))
	})

	It("does not HTML-escape the id into entities", func() {
		Expect(write("a&b")).To(HaveKeyWithValue("id", "a&b"))
	})

	It("responds 200 with a JSON content type", func() {
		write("abc123")
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Header().Get("Content-Type")).To(Equal("application/json"))
	})
})
