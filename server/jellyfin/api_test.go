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
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Router", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = GinkgoT().Context()
	})

	It("serves the public handshake through the mounted handler", func() {
		ds := &tests.MockDataStore{}
		api := New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/System/Info/Public", nil)
		api.ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusOK))
	})

	It("returns 404 JSON for unknown routes", func() {
		api := New(&tests.MockDataStore{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/Nonexistent/Route", nil)
		api.ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusNotFound))
		Expect(w.Header().Get("Content-Type")).To(ContainSubstring("application/json"))
		Expect(w.Body.String()).To(Equal("{}"))
	})

	It("returns 404 JSON for a known path with an unsupported method", func() {
		api := New(&tests.MockDataStore{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPatch, "/System/Info/Public", nil)
		api.ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusNotFound))
		Expect(w.Body.String()).To(Equal("{}"))
	})

	It("registers a player on a general authenticated request, not just playback reports", func() {
		ds := &tests.MockDataStore{}
		auth.Init(ds)
		ur := ds.User().(*tests.MockedUserRepo)
		Expect(ur.Put(ctx, &model.User{ID: testID("u1"), UserName: "alice", NewPassword: "secret"})).To(Succeed())
		token, err := auth.CreateToken(&model.User{ID: testID("u1"), UserName: "alice"})
		Expect(err).ToNot(HaveOccurred())

		fp := &fakePlayers{}
		api := New(ds, nil, nil, nil, fp, nil, nil, nil, nil, nil, nil, nil)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/Users/Me", nil)
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
		api := New(&tests.MockDataStore{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

		login := func() int {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/Users/AuthenticateByName", strings.NewReader(`{"Username":"x","Pw":"y"}`))
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
		ds := &tests.MockDataStore{}
		auth.Init(ds)
		usr := model.User{ID: testID("alice"), UserName: "alice"}
		Expect(ds.User().Put(ctx, &usr)).To(Succeed())
		token, err := auth.CreateAPIToken(&usr, auth.AudienceJellyfin)
		Expect(err).ToNot(HaveOccurred())
		api := New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, quickconnect.New())

		authorize := func() int {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/QuickConnect/Authorize?code=000000", nil)
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
		api := New(&tests.MockDataStore{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		// Every request arrives on the same proxy connection, so only the resolved client IP
		// can separate the buckets.
		handler := middleware.ClientIPFromHeader("X-Real-IP")(api)

		login := func(clientIP string) int {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/Users/AuthenticateByName", strings.NewReader(`{"Username":"x","Pw":"y"}`))
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

var _ = Describe("decodeJSONBody", func() {
	type payload struct {
		Name string `json:"Name"`
	}
	decode := func(body string, limit int64) (*httptest.ResponseRecorder, payload, bool) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		var v payload
		ok := decodeJSONBody(w, r, limit, &v)
		return w, v, ok
	}

	It("decodes one JSON value, allowing trailing whitespace", func() {
		_, v, ok := decode("{\"Name\":\"Mix\"}\r\n\t ", 1024)
		Expect(ok).To(BeTrue())
		Expect(v.Name).To(Equal("Mix"))
	})

	DescribeTable("answers 400 for a body that is not exactly one JSON value",
		func(body string) {
			w, _, ok := decode(body, 1024)
			Expect(ok).To(BeFalse())
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		},
		Entry("empty body", ""),
		Entry("malformed JSON", `{"Name":`),
		Entry("trailing garbage", `{"Name":"Mix"} x`),
		Entry("a second JSON value", `{"Name":"Mix"}{"Name":"Other"}`),
	)

	It("answers 413 for a body over the limit", func() {
		w, _, ok := decode(`{"Name":"`+strings.Repeat("x", 64)+`"}`, 32)
		Expect(ok).To(BeFalse())
		Expect(w.Code).To(Equal(http.StatusRequestEntityTooLarge))
	})
})
