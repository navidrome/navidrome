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
		Expect(decodeProblem(w).Code).To(Equal("not_found"))
	})

	It("returns a 405 problem for a wrong method on a known path", func() {
		w := serve(router, httptest.NewRequest(http.MethodPost, "/api/v1/server", nil))
		Expect(w.Code).To(Equal(http.StatusMethodNotAllowed))
		Expect(decodeProblem(w).Code).To(Equal("method_not_allowed"))
	})

	panicking := func(v any) http.Handler {
		return problemRecoverer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(v) }))
	}

	It("turns a handler panic into a 500 problem", func() {
		w := httptest.NewRecorder()
		panicking("kaboom").ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/boom", nil))
		Expect(w.Code).To(Equal(http.StatusInternalServerError))
		p := decodeProblem(w)
		Expect(p.Code).To(Equal("internal"))
		Expect(p.Detail).To(BeNil())
	})

	It("re-panics http.ErrAbortHandler so the server can drop the connection", func() {
		Expect(func() {
			panicking(http.ErrAbortHandler).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/abort", nil))
		}).To(PanicWith(http.ErrAbortHandler))
	})
})
