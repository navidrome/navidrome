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

	get := func(path string, headers map[string]string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		return serve(router, req)
	}

	DescribeTable("serves the embedded bundle",
		func(path, contentType string, body []byte) {
			w := get(path, nil)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(Equal(contentType))
			Expect(w.Header().Get("ETag")).ToNot(BeEmpty())
			Expect(w.Header().Get("Cache-Control")).To(Equal("no-cache"))
			Expect(w.Body.Bytes()).To(Equal(body))
		},
		Entry("JSON", "/api/v1/openapi.json", "application/json", api.SpecJSON()),
		Entry("YAML", "/api/v1/openapi.yaml", "application/yaml", api.SpecYAML()),
	)

	DescribeTable("revalidates with If-None-Match",
		func(ifNoneMatch func(etag string) string, expected int) {
			etag := get("/api/v1/openapi.json", nil).Header().Get("ETag")
			w := get("/api/v1/openapi.json", map[string]string{"If-None-Match": ifNoneMatch(etag)})
			Expect(w.Code).To(Equal(expected))
			if expected == http.StatusNotModified {
				Expect(w.Body.Len()).To(BeZero())
				Expect(w.Header().Get("ETag")).To(Equal(etag))
			} else {
				Expect(w.Body.Bytes()).To(Equal(api.SpecJSON()))
			}
		},
		Entry("exact ETag", func(etag string) string { return etag }, http.StatusNotModified),
		Entry("ETag in a list", func(etag string) string { return `"other", ` + etag }, http.StatusNotModified),
		Entry("weak ETag", func(etag string) string { return "W/" + etag }, http.StatusNotModified),
		Entry("wildcard", func(string) string { return "*" }, http.StatusNotModified),
		Entry("stale ETag", func(string) string { return `"stale"` }, http.StatusOK),
	)

	It("ignores Range and returns the full document", func() {
		w := get("/api/v1/openapi.json", map[string]string{"Range": "bytes=0-9"})
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Header().Get("Accept-Ranges")).To(BeEmpty())
		Expect(w.Body.Bytes()).To(Equal(api.SpecJSON()))
	})

	It("uses different ETags for JSON and YAML", func() {
		j := get("/api/v1/openapi.json", nil)
		y := get("/api/v1/openapi.yaml", nil)
		Expect(j.Header().Get("ETag")).ToNot(Equal(y.Header().Get("ETag")))
	})
})
