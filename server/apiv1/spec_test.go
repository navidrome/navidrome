package apiv1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/api"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
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

	Describe("with a base path", func() {
		decode := func(format string, body []byte) map[string]any {
			var doc map[string]any
			if format == "json" {
				ExpectWithOffset(1, json.Unmarshal(body, &doc)).To(Succeed())
			} else {
				ExpectWithOffset(1, yaml.Unmarshal(body, &doc)).To(Succeed())
			}
			return doc
		}
		serverURL := func(doc map[string]any) any {
			return doc["servers"].([]any)[0].(map[string]any)["url"]
		}
		var plainETag string

		BeforeEach(func() {
			plainETag = get("/api/v1/openapi.json", nil).Header().Get("ETag")
			DeferCleanup(configtest.SetupConfig())
			conf.Server.BasePath = "/music"
			router = New(&tests.MockDataStore{})
		})

		DescribeTable("advertises the server under the base path and changes nothing else",
			func(path, format string, bundle []byte) {
				w := get(path, nil)
				Expect(w.Code).To(Equal(http.StatusOK))
				served, original := decode(format, w.Body.Bytes()), decode(format, bundle)
				Expect(serverURL(served)).To(Equal("/music/api/v1"))
				delete(served, "servers")
				delete(original, "servers")
				Expect(served).To(Equal(original))
			},
			Entry("JSON", "/api/v1/openapi.json", "json", api.SpecJSON()),
			Entry("YAML", "/api/v1/openapi.yaml", "yaml", api.SpecYAML()),
		)

		It("uses its own ETag, and still revalidates", func() {
			etag := get("/api/v1/openapi.json", nil).Header().Get("ETag")
			Expect(etag).ToNot(Equal(plainETag))
			Expect(get("/api/v1/openapi.json", map[string]string{"If-None-Match": etag}).Code).To(Equal(http.StatusNotModified))
		})

		It("escapes base paths that need quoting", func() {
			conf.Server.BasePath = "/my music: \"live\""
			router = New(&tests.MockDataStore{})
			Expect(serverURL(decode("json", get("/api/v1/openapi.json", nil).Body.Bytes()))).To(Equal("/my music: \"live\"/api/v1"))
			Expect(serverURL(decode("yaml", get("/api/v1/openapi.yaml", nil).Body.Bytes()))).To(Equal("/my music: \"live\"/api/v1"))
		})
	})

	DescribeTable("the bundle advertises the API path exactly once, which the base-path rewrite relies on",
		func(bundle []byte) {
			Expect(bytes.Count(bundle, []byte(consts.URLPathAPIv1))).To(Equal(1))
		},
		Entry("JSON", api.SpecJSON()),
		Entry("YAML", api.SpecYAML()),
	)
})
