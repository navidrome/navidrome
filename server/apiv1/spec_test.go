package apiv1

import (
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/api"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("OpenAPI document routes", func() {
	var router *Router

	BeforeEach(func() {
		router = New(&tests.MockDataStore{})
	})

	It("serves the JSON bundle with an ETag", func() {
		w := serve(router, httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil))
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Header().Get("Content-Type")).To(HavePrefix("application/json"))
		Expect(w.Header().Get("ETag")).ToNot(BeEmpty())
		Expect(w.Header().Get("Cache-Control")).To(Equal("no-cache"))
		Expect(w.Body.Bytes()).To(Equal(api.SpecJSON()))
	})

	It("serves the YAML bundle", func() {
		w := serve(router, httptest.NewRequest(http.MethodGet, "/api/v1/openapi.yaml", nil))
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Header().Get("Content-Type")).To(HavePrefix("application/yaml"))
		Expect(w.Body.Bytes()).To(Equal(api.SpecYAML()))
	})

	It("answers 304 when If-None-Match matches", func() {
		first := serve(router, httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil))
		req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil)
		req.Header.Set("If-None-Match", first.Header().Get("ETag"))
		w := serve(router, req)
		Expect(w.Code).To(Equal(http.StatusNotModified))
		Expect(w.Body.Len()).To(BeZero())
	})

	It("ignores Range and returns the full document", func() {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil)
		req.Header.Set("Range", "bytes=0-9")
		w := serve(router, req)
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Header().Get("Accept-Ranges")).To(BeEmpty())
		Expect(w.Body.Bytes()).To(Equal(api.SpecJSON()))
	})

	It("answers 304 when If-None-Match lists the ETag", func() {
		first := serve(router, httptest.NewRequest(http.MethodGet, "/api/v1/openapi.yaml", nil))
		req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.yaml", nil)
		req.Header.Set("If-None-Match", `"other", `+first.Header().Get("ETag"))
		w := serve(router, req)
		Expect(w.Code).To(Equal(http.StatusNotModified))
		Expect(w.Body.Len()).To(BeZero())
	})

	It("answers 304 when If-None-Match is *", func() {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil)
		req.Header.Set("If-None-Match", "*")
		w := serve(router, req)
		Expect(w.Code).To(Equal(http.StatusNotModified))
		Expect(w.Body.Len()).To(BeZero())
	})

	It("returns the full document when If-None-Match does not match", func() {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil)
		req.Header.Set("If-None-Match", `"stale"`)
		w := serve(router, req)
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Body.Bytes()).To(Equal(api.SpecJSON()))
	})

	It("uses different ETags for JSON and YAML", func() {
		j := serve(router, httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil))
		y := serve(router, httptest.NewRequest(http.MethodGet, "/api/v1/openapi.yaml", nil))
		Expect(j.Header().Get("ETag")).ToNot(Equal(y.Header().Get("ETag")))
	})
})
