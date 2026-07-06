package jellyfin

import (
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("caseInsensitivePaths", func() {
	var handler http.Handler
	var gotID string

	var gotContainer string

	BeforeEach(func() {
		gotID = ""
		gotContainer = ""
		r := chi.NewRouter()
		r.Get("/Foo/{id}/Bar", func(w http.ResponseWriter, req *http.Request) {
			gotID = chi.URLParam(req, "id")
			w.WriteHeader(http.StatusOK)
		})
		// A mixed literal+param segment (like Jellyfin's /Audio/{id}/stream.{container}): the "stream"
		// literal prefix is registered separately via the bare /Foo/{id}/stream route below.
		r.Get("/Foo/{id}/stream", func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		r.Get("/Foo/{id}/stream.{container}", func(w http.ResponseWriter, req *http.Request) {
			gotContainer = chi.URLParam(req, "container")
			w.WriteHeader(http.StatusOK)
		})
		handler = caseInsensitivePaths(r)
	})

	It("normalizes the literal prefix of a mixed literal.param segment", func() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/foo/ID/STREAM.mp3", nil)
		handler.ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(gotContainer).To(Equal("mp3"))
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

	It("normalizes the literal prefix of a mixed literal.extension segment", func() {
		canon := map[string]string{"audio": "Audio", "stream": "stream"}
		got := normalizeCase("/audio/XyZ123NotARoute/STREAM.MP3", canon)
		Expect(got).To(Equal("/Audio/XyZ123NotARoute/stream.mp3"))
	})

	It("leaves a dotted segment untouched when its prefix isn't a known literal", func() {
		canon := map[string]string{"audio": "Audio"}
		got := normalizeCase("/audio/some.file.id", canon)
		Expect(got).To(Equal("/Audio/some.file.id"))
	})
})
