package apiv1

import (
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Router", func() {
	var router *Router

	BeforeEach(func() {
		router = New(&tests.MockDataStore{})
	})

	It("returns a 404 problem for unknown paths", func() {
		w := serve(router, httptest.NewRequest(http.MethodGet, "/api/v1/nope", nil))
		Expect(w.Code).To(Equal(http.StatusNotFound))
		Expect(w.Header().Get("Content-Type")).To(Equal(problemContentType))
		Expect(decodeProblem(w).Code).To(Equal(ProblemCodeNotFound))
	})

	It("returns a 405 problem listing the allowed methods for a wrong method on a known path", func() {
		w := serve(router, httptest.NewRequest(http.MethodPost, "/api/v1/server", nil))
		Expect(w.Code).To(Equal(http.StatusMethodNotAllowed))
		Expect(w.Header().Get("Allow")).To(Equal("GET, HEAD"))
		Expect(decodeProblem(w).Code).To(Equal(ProblemCodeMethodNotAllowed))
	})

	DescribeTable("answers HEAD wherever GET is routed",
		func(path, contentType string) {
			w := serve(router, httptest.NewRequest(http.MethodHead, path, nil))
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(Equal(contentType))
		},
		Entry("server info", "/api/v1/server", "application/json"),
		Entry("JSON spec", "/api/v1/openapi.json", "application/json"),
		Entry("YAML spec", "/api/v1/openapi.yaml", "application/yaml"),
	)

	It("revalidates HEAD requests with If-None-Match", func() {
		etag := serve(router, httptest.NewRequest(http.MethodHead, "/api/v1/openapi.json", nil)).Header().Get("ETag")
		req := httptest.NewRequest(http.MethodHead, "/api/v1/openapi.json", nil)
		req.Header.Set("If-None-Match", etag)
		Expect(serve(router, req).Code).To(Equal(http.StatusNotModified))
	})

	It("returns a 404 problem for HEAD on unknown paths", func() {
		w := serve(router, httptest.NewRequest(http.MethodHead, "/api/v1/nope", nil))
		Expect(w.Code).To(Equal(http.StatusNotFound))
		Expect(w.Header().Get("Allow")).To(BeEmpty())
	})

	panicking := func(v any) http.Handler {
		return problemRecoverer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(v) }))
	}

	It("turns a handler panic into a 500 problem", func() {
		w := httptest.NewRecorder()
		panicking("kaboom").ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/boom", nil))
		Expect(w.Code).To(Equal(http.StatusInternalServerError))
		p := decodeProblem(w)
		Expect(p.Code).To(Equal(ProblemCodeInternal))
		Expect(p.Detail).To(BeNil())
	})

	It("re-panics http.ErrAbortHandler so the server can drop the connection", func() {
		Expect(func() {
			panicking(http.ErrAbortHandler).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/abort", nil))
		}).To(PanicWith(http.ErrAbortHandler))
	})
})
