package jellyfin

import (
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("decodeJSONBody", func() {
	type payload struct {
		Name string `json:"Name"`
	}
	decode := func(body string, limit int64) (*httptest.ResponseRecorder, payload, bool) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		var v payload
		ok := decodeJSONBody(w, r, limit, &v)
		return w, v, ok
	}

	It("decodes one JSON value, allowing trailing whitespace", func() {
		_, v, ok := decode("{\"Name\":\"Mix\"}\r\n\t ", 1024)
		Expect(ok).To(BeTrue())
		Expect(v.Name).To(Equal("Mix"))
	})

	DescribeTable("answers 400 for a body that is not exactly one JSON value",
		func(body string) {
			w, _, ok := decode(body, 1024)
			Expect(ok).To(BeFalse())
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		},
		Entry("empty body", ""),
		Entry("malformed JSON", `{"Name":`),
		Entry("trailing garbage", `{"Name":"Mix"} x`),
		Entry("a second JSON value", `{"Name":"Mix"}{"Name":"Other"}`),
	)

	It("answers 413 for a body over the limit", func() {
		w, _, ok := decode(`{"Name":"`+strings.Repeat("x", 64)+`"}`, 32)
		Expect(ok).To(BeFalse())
		Expect(w.Code).To(Equal(http.StatusRequestEntityTooLarge))
	})
})
