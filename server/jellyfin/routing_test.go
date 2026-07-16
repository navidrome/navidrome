package jellyfin

import (
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Real Jellyfin servers route path segments case-insensitively, but chi's default matching is
// case-sensitive. Jellyfin wires up server.CaseInsensitivePaths (see server/case_insensitive_routes.go
// for the unit-level tests of that helper) to work around this. These tests are an end-to-end proof
// that requests using non-canonical casing are still routed correctly, both when the router is used
// directly and when mounted under a parent (as it is in production via server.MountRouter).
var _ = Describe("Case-insensitive routing", func() {
	var api *Router

	BeforeEach(func() {
		api = New(&tests.MockDataStore{}, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	})

	It("serves a fully lowercase path directly", func() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/system/info/public", nil)
		api.ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusOK))
	})

	It("serves a mixed/weird-case path directly", func() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/SYSTEM/Info/PUBLIC", nil)
		api.ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusOK))
	})

	It("serves a lowercase login path directly", func() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/users/authenticatebyname", nil)
		api.ServeHTTP(w, r)
		// MockDataStore has no users, so authentication itself may fail downstream, but the
		// route must be found (not a 404) to prove case-insensitive matching worked.
		Expect(w.Code).ToNot(Equal(http.StatusNotFound))
	})

	It("serves a lowercase path when mounted under a parent router, replicating production", func() {
		parent := chi.NewRouter()
		parent.Mount("/jellyfin", api)
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/jellyfin/system/info/public", nil)
		parent.ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusOK))
	})
})
