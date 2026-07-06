package jellyfin

import (
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Router", func() {
	It("serves the public handshake through the mounted handler", func() {
		ds := &tests.MockDataStore{}
		api := New(ds, nil, nil, nil, nil, nil, nil, nil)
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/System/Info/Public", nil)
		api.ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusOK))
	})

	It("returns 404 JSON for unknown routes", func() {
		api := New(&tests.MockDataStore{}, nil, nil, nil, nil, nil, nil, nil)
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/Nonexistent/Route", nil)
		api.ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusNotFound))
		Expect(w.Header().Get("Content-Type")).To(ContainSubstring("application/json"))
		Expect(w.Body.String()).To(Equal("{}"))
	})

	It("returns 404 JSON for a known path with an unsupported method", func() {
		api := New(&tests.MockDataStore{}, nil, nil, nil, nil, nil, nil, nil)
		w := httptest.NewRecorder()
		r := httptest.NewRequest("PATCH", "/System/Info/Public", nil)
		api.ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusNotFound))
		Expect(w.Body.String()).To(Equal("{}"))
	})
})
