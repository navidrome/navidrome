package jellyfin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
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

	It("rate-limits AuthenticateByName by IP when a login limit is configured", func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.AuthRequestLimit = 2
		conf.Server.AuthWindowLength = time.Minute
		api := New(&tests.MockDataStore{}, nil, nil, nil, nil, nil, nil, nil)

		login := func() int {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/AuthenticateByName", strings.NewReader(`{"Username":"x","Pw":"y"}`))
			r.RemoteAddr = "10.0.0.1:1234"
			api.ServeHTTP(w, r)
			return w.Code
		}
		// The bad credentials would be 401; the limiter cuts in on the 3rd attempt with 429.
		Expect(login()).To(Equal(http.StatusUnauthorized))
		Expect(login()).To(Equal(http.StatusUnauthorized))
		Expect(login()).To(Equal(http.StatusTooManyRequests))
	})
})
