package server

import (
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CaseInsensitivePaths", func() {
	var handler http.Handler
	var gotID string

	BeforeEach(func() {
		gotID = ""
		r := chi.NewRouter()
		r.Get("/Foo/{id}/Bar", func(w http.ResponseWriter, req *http.Request) {
			gotID = chi.URLParam(req, "id")
			w.WriteHeader(http.StatusOK)
		})
		handler = CaseInsensitivePaths(r)
	})

	It("matches a lower-cased request path against mixed-case registered literals", func() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/foo/ID/bar", nil)
		handler.ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusOK))
	})

	It("preserves the id segment's original casing", func() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/foo/ID/bar", nil)
		handler.ServeHTTP(w, r)
		Expect(gotID).To(Equal("ID"))
	})

	It("leaves a real mixed-case id untouched while still matching literals", func() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/foo/cjsFeXbNOaaSjASu3DM93g/bar", nil)
		handler.ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(gotID).To(Equal("cjsFeXbNOaaSjASu3DM93g"))
	})
})

var _ = Describe("normalizeCase", func() {
	It("rewrites known literal segments to their canonical case", func() {
		canon := map[string]string{
			"audio":  "Audio",
			"stream": "stream",
		}
		got := normalizeCase("/audio/XyZ123NotARoute/STREAM", canon)
		Expect(got).To(Equal("/Audio/XyZ123NotARoute/stream"))
	})
})
