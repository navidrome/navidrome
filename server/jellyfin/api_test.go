package jellyfin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/quickconnect"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Router", func() {
	var ctx context.Context
	var ds model.DataStore

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = persistence.New(db.Db())
	})

	It("serves the public handshake through the mounted handler", func() {
		api := New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		w := httptest.NewRecorder()
		r := httptest.NewRequestWithContext(ctx, http.MethodGet, "/System/Info/Public", nil)
		api.ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusOK))
	})

	It("returns 404 JSON for unknown routes", func() {
		api := New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		w := httptest.NewRecorder()
		r := httptest.NewRequestWithContext(ctx, http.MethodGet, "/Nonexistent/Route", nil)
		api.ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusNotFound))
		Expect(w.Header().Get("Content-Type")).To(ContainSubstring("application/json"))
		Expect(w.Body.String()).To(Equal("{}"))
	})

	It("returns 404 JSON for a known path with an unsupported method", func() {
		api := New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		w := httptest.NewRecorder()
		r := httptest.NewRequestWithContext(ctx, http.MethodPatch, "/System/Info/Public", nil)
		api.ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusNotFound))
		Expect(w.Body.String()).To(Equal("{}"))
	})

	It("registers a player on a general authenticated request, not just playback reports", func() {
		auth.Init(ds)
		Expect(ds.User().Put(ctx, &model.User{ID: testID("u1"), UserName: "alice", NewPassword: "secret"})).To(Succeed())
		token, err := auth.CreateToken(&model.User{ID: testID("u1"), UserName: "alice"})
		Expect(err).ToNot(HaveOccurred())

		fp := &fakePlayers{}
		api := New(ds, nil, nil, nil, fp, nil, nil, nil, nil, nil, nil, nil)

		w := httptest.NewRecorder()
		r := httptest.NewRequestWithContext(ctx, http.MethodGet, "/Users/Me", nil)
		r.Header.Set("X-Emby-Authorization", `MediaBrowser Client="Jellify", Device="Phone", DeviceId="dev-1", Version="1.0"`)
		r.Header.Set("X-Emby-Token", token)
		api.ServeHTTP(w, r)

		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(fp.registerCalls).To(Equal(1))
		Expect(fp.lastClient).To(Equal("Jellify"))
	})

	It("rate-limits AuthenticateByName by IP when a login limit is configured", func() {
		conf.Update(func(c *conf.Options) {
			c.AuthRequestLimit = 2
			c.AuthWindowLength = time.Minute
		})
		api := New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

		login := func() int {
			w := httptest.NewRecorder()
			r := httptest.NewRequestWithContext(ctx, http.MethodPost, "/Users/AuthenticateByName", strings.NewReader(`{"Username":"x","Pw":"y"}`))
			r.RemoteAddr = "10.0.0.1:1234"
			api.ServeHTTP(w, r)
			return w.Code
		}
		// The bad credentials would be 401; the limiter cuts in on the 3rd attempt with 429.
		Expect(login()).To(Equal(http.StatusUnauthorized))
		Expect(login()).To(Equal(http.StatusUnauthorized))
		Expect(login()).To(Equal(http.StatusTooManyRequests))
	})

	It("rate-limits Quick Connect approval by IP when a login limit is configured", func() {
		conf.Update(func(c *conf.Options) {
			c.AuthRequestLimit = 2
			c.AuthWindowLength = time.Minute
			c.Jellyfin.QuickConnect = true
		})
		auth.Init(ds)
		usr := model.User{ID: testID("alice"), UserName: "alice"}
		Expect(ds.User().Put(ctx, &usr)).To(Succeed())
		token, err := auth.CreateAPIToken(&usr, auth.AudienceJellyfin)
		Expect(err).ToNot(HaveOccurred())
		api := New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, quickconnect.New())

		authorize := func() int {
			w := httptest.NewRecorder()
			r := httptest.NewRequestWithContext(ctx, http.MethodPost, "/QuickConnect/Authorize?code=000000", nil)
			r.RemoteAddr = "10.0.0.1:1234"
			r.Header.Set("X-Emby-Token", token)
			api.ServeHTTP(w, r)
			return w.Code
		}
		// An unknown code is 404; the limiter cuts in on the 3rd attempt with 429.
		Expect(authorize()).To(Equal(http.StatusNotFound))
		Expect(authorize()).To(Equal(http.StatusNotFound))
		Expect(authorize()).To(Equal(http.StatusTooManyRequests))
	})

	It("rate-limits AuthenticateByName by resolved client IP, not by the proxy connection", func() {
		conf.Update(func(c *conf.Options) {
			c.AuthRequestLimit = 1
			c.AuthWindowLength = time.Minute
		})
		api := New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		// Every request arrives on the same proxy connection, so only the resolved client IP
		// can separate the buckets.
		handler := middleware.ClientIPFromHeader("X-Real-IP")(api)

		login := func(clientIP string) int {
			w := httptest.NewRecorder()
			r := httptest.NewRequestWithContext(ctx, http.MethodPost, "/Users/AuthenticateByName", strings.NewReader(`{"Username":"x","Pw":"y"}`))
			r.RemoteAddr = "10.0.0.1:1234"
			r.Header.Set("X-Real-IP", clientIP)
			handler.ServeHTTP(w, r)
			return w.Code
		}
		Expect(login("203.0.113.1")).To(Equal(http.StatusUnauthorized))
		Expect(login("203.0.113.1")).To(Equal(http.StatusTooManyRequests))
		Expect(login("203.0.113.2")).To(Equal(http.StatusUnauthorized))
	})
})
