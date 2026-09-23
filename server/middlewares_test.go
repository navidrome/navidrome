package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/publicurl"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("middlewares", func() {
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
	})
	Describe("robotsTXT", func() {
		var nextCalled bool
		next := func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
		}
		BeforeEach(func() {
			nextCalled = false
		})

		It("returns the robot.txt when requested from root", func() {
			r := httptest.NewRequest("GET", "/robots.txt", nil)
			w := httptest.NewRecorder()

			robotsTXT(os.DirFS("tests/fixtures"))(http.HandlerFunc(next)).ServeHTTP(w, r)

			Expect(nextCalled).To(BeFalse())
			Expect(w.Body.String()).To(HavePrefix("User-agent:"))
		})

		It("allows prefixes", func() {
			r := httptest.NewRequest("GET", "/app/robots.txt", nil)
			w := httptest.NewRecorder()

			robotsTXT(os.DirFS("tests/fixtures"))(http.HandlerFunc(next)).ServeHTTP(w, r)

			Expect(nextCalled).To(BeFalse())
			Expect(w.Body.String()).To(HavePrefix("User-agent:"))
		})

		It("passes through requests for other files", func() {
			r := httptest.NewRequest("GET", "/this_is_not_a_robots.txt_file", nil)
			w := httptest.NewRecorder()

			robotsTXT(os.DirFS("tests/fixtures"))(http.HandlerFunc(next)).ServeHTTP(w, r)

			Expect(nextCalled).To(BeTrue())
		})
	})

	Describe("serverAddressMiddleware", func() {
		var (
			nextHandler http.Handler
			middleware  http.Handler
			recorder    *httptest.ResponseRecorder
			req         *http.Request
			gotScheme   string
			gotHost     string
			gotOK       bool
		)

		BeforeEach(func() {
			gotScheme, gotHost, gotOK = "", "", false
			nextHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotScheme, gotHost, gotOK = request.ServerAddressFrom(r.Context())
				w.WriteHeader(http.StatusOK)
			})
			middleware = serverAddressMiddleware(nextHandler)
			recorder = httptest.NewRecorder()
		})

		Context("with no X-Forwarded headers", func() {
			BeforeEach(func() {
				req, _ = http.NewRequest("GET", "http://example.com", nil)
			})

			It("should not modify the request", func() {
				middleware.ServeHTTP(recorder, req)
				Expect(req.Host).To(Equal("example.com"))
				Expect(req.URL.Scheme).To(Equal("http"))
			})

			It("should record the address in the context", func() {
				middleware.ServeHTTP(recorder, req)
				Expect(gotOK).To(BeTrue())
				Expect(gotScheme).To(Equal("http"))
				Expect(gotHost).To(Equal("example.com"))
			})
		})

		Context("with X-Forwarded-Host header", func() {
			BeforeEach(func() {
				req, _ = http.NewRequest("GET", "http://example.com", nil)
				req.Header.Set("X-Forwarded-Host", "forwarded.example.com")
			})

			It("should modify the request with the X-Forwarded-Host header value", func() {
				middleware.ServeHTTP(recorder, req)
				Expect(req.Host).To(Equal("forwarded.example.com"))
				Expect(req.URL.Scheme).To(Equal("http"))
			})
		})

		Context("with X-Forwarded-Proto header", func() {
			BeforeEach(func() {
				req, _ = http.NewRequest("GET", "http://example.com", nil)
				req.Header.Set("X-Forwarded-Proto", "https")
			})

			It("should modify the request with the X-Forwarded-Proto header value", func() {
				middleware.ServeHTTP(recorder, req)
				Expect(req.Host).To(Equal("example.com"))
				Expect(req.URL.Scheme).To(Equal("https"))
			})
		})

		Context("with X-Forwarded-Scheme header", func() {
			BeforeEach(func() {
				req, _ = http.NewRequest("GET", "http://example.com", nil)
				req.Header.Set("X-Forwarded-Scheme", "https")
			})

			It("should modify the request with the X-Forwarded-Scheme header value", func() {
				middleware.ServeHTTP(recorder, req)
				Expect(req.Host).To(Equal("example.com"))
				Expect(req.URL.Scheme).To(Equal("https"))
			})
		})

		Context("with multiple X-Forwarded headers", func() {
			BeforeEach(func() {
				req, _ = http.NewRequest("GET", "http://example.com", nil)
				req.Header.Set("X-Forwarded-Host", "forwarded.example.com")
				req.Header.Set("X-Forwarded-Proto", "https")
				req.Header.Set("X-Forwarded-Scheme", "http")
			})

			It("should modify the request with the first non-empty X-Forwarded header value", func() {
				middleware.ServeHTTP(recorder, req)
				Expect(req.Host).To(Equal("forwarded.example.com"))
				Expect(req.URL.Scheme).To(Equal("https"))
			})

			It("should record the forwarded address in the context", func() {
				middleware.ServeHTTP(recorder, req)
				Expect(gotOK).To(BeTrue())
				Expect(gotScheme).To(Equal("https"))
				Expect(gotHost).To(Equal("forwarded.example.com"))
			})

			It("lets a handler build a public URL on the forwarded address", func() {
				var got string
				serverAddressMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					got = publicurl.AbsoluteURL(r.Context(), "/share/img/token", nil)
				})).ServeHTTP(recorder, req)

				Expect(got).To(Equal("https://forwarded.example.com/share/img/token"))
			})
		})

		Context("with multiple values in X-Forwarded-Host header", func() {
			BeforeEach(func() {
				req, _ = http.NewRequest("GET", "http://example.com", nil)
				req.Header.Set("X-Forwarded-Host", "forwarded1.example.com, forwarded2.example.com")
			})

			It("should modify the request with the first value in X-Forwarded-Host header", func() {
				middleware.ServeHTTP(recorder, req)
				Expect(req.Host).To(Equal("forwarded1.example.com"))
				Expect(req.URL.Scheme).To(Equal("http"))
			})
		})
	})

	Describe("clientUniqueIDMiddleware", func() {
		var (
			nextHandler http.Handler
			middleware  http.Handler
			req         *http.Request
			nextReq     *http.Request
			rec         *httptest.ResponseRecorder
		)

		BeforeEach(func() {
			nextHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				nextReq = r
			})
			middleware = clientUniqueIDMiddleware(nextHandler)
			req, _ = http.NewRequest(http.MethodGet, "/", nil)
			rec = httptest.NewRecorder()
		})

		Context("when the request header has the unique client ID", func() {
			BeforeEach(func() {
				req.Header.Set(consts.UIClientUniqueIDHeader, "123456")
				conf.Server.BasePath = "/music"
			})

			It("sets the unique client ID as a cookie and adds it to the request context", func() {
				middleware.ServeHTTP(rec, req)

				Expect(rec.Result().Cookies()).To(HaveLen(1))
				Expect(rec.Result().Cookies()[0].Name).To(Equal(consts.UIClientUniqueIDHeader))
				Expect(rec.Result().Cookies()[0].Value).To(Equal("123456"))
				Expect(rec.Result().Cookies()[0].MaxAge).To(Equal(consts.CookieExpiry))
				Expect(rec.Result().Cookies()[0].HttpOnly).To(BeTrue())
				Expect(rec.Result().Cookies()[0].Secure).To(BeTrue())
				Expect(rec.Result().Cookies()[0].SameSite).To(Equal(http.SameSiteStrictMode))
				Expect(rec.Result().Cookies()[0].Path).To(Equal("/music"))
				clientUniqueId, _ := request.ClientUniqueIdFrom(nextReq.Context())
				Expect(clientUniqueId).To(Equal("123456"))
			})
		})

		Context("when the request header does not have the unique client ID", func() {
			Context("when the request has the unique client ID in a cookie", func() {
				BeforeEach(func() {
					req.AddCookie(&http.Cookie{
						Name:  consts.UIClientUniqueIDHeader,
						Value: "123456",
					})
				})

				It("adds the unique client ID to the request context", func() {
					middleware.ServeHTTP(rec, req)

					Expect(rec.Result().Cookies()).To(HaveLen(0))

					clientUniqueId, _ := request.ClientUniqueIdFrom(nextReq.Context())
					Expect(clientUniqueId).To(Equal("123456"))
				})
			})

			Context("when the request does not have the unique client ID in a cookie", func() {
				It("does not add the unique client ID to the request context", func() {
					middleware.ServeHTTP(rec, req)

					Expect(rec.Result().Cookies()).To(HaveLen(0))

					clientUniqueId, _ := request.ClientUniqueIdFrom(nextReq.Context())
					Expect(clientUniqueId).To(BeEmpty())
				})
			})
		})
	})

	Describe("URLParamsMiddleware", func() {
		var (
			router      *chi.Mux
			middleware  http.Handler
			recorder    *httptest.ResponseRecorder
			testHandler http.HandlerFunc
		)

		BeforeEach(func() {
			router = chi.NewRouter()
			recorder = httptest.NewRecorder()
			testHandler = func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("OK"))
			}
		})

		Context("when request has no query parameters", func() {
			It("adds URL parameters to the request", func() {
				middleware = URLParamsMiddleware(testHandler)
				router.Mount("/", middleware)

				req, _ := http.NewRequest("GET", "/?user=1", nil)
				router.ServeHTTP(recorder, req)

				Expect(recorder.Code).To(Equal(http.StatusOK))
				Expect(recorder.Body.String()).To(Equal("OK"))
				Expect(req.URL.RawQuery).To(ContainSubstring("user=1"))
			})
		})

		Context("when request has query parameters", func() {
			It("merges URL parameters and query parameters", func() {
				router.Route("/{key}", func(r chi.Router) {
					r.Use(URLParamsMiddleware)
					r.Get("/", testHandler)
				})

				req, _ := http.NewRequest("GET", "/test?key=value", nil)
				router.ServeHTTP(recorder, req)
				Expect(recorder.Code).To(Equal(http.StatusOK))
				Expect(recorder.Body.String()).To(Equal("OK"))
				Expect(req.URL.RawQuery).To(ContainSubstring("key=value"))
				Expect(req.URL.RawQuery).To(ContainSubstring("%3Akey=test"))
			})
		})

		Context("when URL parameter has wildcard key", func() {
			It("does not include wildcard key in query parameters", func() {
				router.Route("/{t*}", func(r chi.Router) {
					r.Use(URLParamsMiddleware)
					r.Get("/", testHandler)
				})

				req, _ := http.NewRequest("GET", "/test?key=value", nil)
				router.ServeHTTP(recorder, req)

				Expect(recorder.Code).To(Equal(http.StatusOK))
				Expect(recorder.Body.String()).To(Equal("OK"))
				Expect(req.URL.RawQuery).To(ContainSubstring("key=value"))
			})
		})

		Context("when URL parameters require encoding", func() {
			It("encodes URL parameters correctly", func() {
				router.Route("/{key}", func(r chi.Router) {
					r.Use(URLParamsMiddleware)
					r.Get("/", testHandler)
				})

				req, _ := http.NewRequest("GET", "/test with space?key=another value", nil)
				router.ServeHTTP(recorder, req)

				Expect(recorder.Code).To(Equal(http.StatusOK))
				Expect(recorder.Body.String()).To(Equal("OK"))
				queryValues, _ := url.ParseQuery(req.URL.RawQuery)
				Expect(queryValues.Get(":key")).To(Equal("test with space"))
				Expect(queryValues.Get("key")).To(Equal("another value"))
			})
		})

		Context("when there are multiple URL parameters", func() {
			It("includes all URL parameters in the query string", func() {
				router.Route("/{key}/{value}", func(r chi.Router) {
					r.Use(URLParamsMiddleware)
					r.Get("/", testHandler)
				})

				req, _ := http.NewRequest("GET", "/test/value?key=other_value", nil)
				router.ServeHTTP(recorder, req)

				Expect(recorder.Code).To(Equal(http.StatusOK))
				Expect(recorder.Body.String()).To(Equal("OK"))

				queryValues, _ := url.ParseQuery(req.URL.RawQuery)
				Expect(queryValues.Get(":key")).To(Equal("test"))
				Expect(queryValues.Get(":value")).To(Equal("value"))
				Expect(queryValues.Get("key")).To(Equal("other_value"))
			})
		})
	})

	Describe("UpdateLastAccessMiddleware", func() {
		var (
			middleware     func(next http.Handler) http.Handler
			req            *http.Request
			ctx            context.Context
			ds             *tests.MockDataStore
			id             string
			lastAccessTime time.Time
		)

		callMiddleware := func(req *http.Request) {
			middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(nil, req)
		}

		BeforeEach(func() {
			id = uuid.NewString()
			ds = &tests.MockDataStore{}
			lastAccessTime = time.Now()
			Expect(ds.User(ctx).Put(&model.User{ID: id, UserName: "johndoe", LastAccessAt: &lastAccessTime})).
				To(Succeed())

			middleware = UpdateLastAccessMiddleware(ds)
			ctx = request.WithUser(
				context.Background(),
				model.User{ID: id, UserName: "johndoe"},
			)
			req, _ = http.NewRequest(http.MethodGet, "/", nil)
			req = req.WithContext(ctx)
		})

		Context("when the request has a user", func() {
			It("does calls the next handler", func() {
				called := false
				middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					called = true
				})).ServeHTTP(nil, req)
				Expect(called).To(BeTrue())
			})

			It("updates the last access time", func() {
				time.Sleep(3 * time.Millisecond)

				callMiddleware(req)

				user, _ := ds.MockedUser.FindByUsername("johndoe")
				Expect(*user.LastAccessAt).To(BeTemporally(">", lastAccessTime, time.Second))
			})

			It("skip fast successive requests", func() {
				// First request
				callMiddleware(req)
				user, _ := ds.MockedUser.FindByUsername("johndoe")
				lastAccessTime = *user.LastAccessAt // Store the last access time

				// Second request
				time.Sleep(3 * time.Millisecond)
				callMiddleware(req)

				// The second request should not have changed the last access time
				user, _ = ds.MockedUser.FindByUsername("johndoe")
				Expect(user.LastAccessAt).To(Equal(&lastAccessTime))
			})
		})
		Context("when the request has no user", func() {
			It("does not update the last access time", func() {
				req = req.WithContext(context.Background())
				callMiddleware(req)

				usr, _ := ds.MockedUser.FindByUsername("johndoe")
				Expect(usr.LastAccessAt).To(Equal(&lastAccessTime))
			})
		})
	})
	Describe("realIPMiddleware", func() {
		var resolved, remoteAddr string
		var proxyIP any
		next := func(w http.ResponseWriter, r *http.Request) {
			resolved = middleware.GetClientIP(r.Context())
			remoteAddr = r.RemoteAddr
			proxyIP = r.Context().Value(request.ReverseProxyIp)
		}
		call := func(peer string, headers map[string]string) {
			resolved, remoteAddr, proxyIP = "", "", nil
			r := httptest.NewRequest("POST", "/auth/login", nil)
			r.RemoteAddr = peer
			for k, v := range headers {
				r.Header.Set(k, v)
			}
			realIPMiddleware(http.HandlerFunc(next)).ServeHTTP(httptest.NewRecorder(), r)
		}

		Context("without a trusted proxy", func() {
			It("ignores client-supplied forwarding headers", func() {
				call("10.0.0.1:1234", map[string]string{
					"X-Forwarded-For": "203.0.113.5",
					"X-Real-IP":       "203.0.113.6",
					"True-Client-IP":  "203.0.113.7",
				})
				Expect(resolved).To(Equal("10.0.0.1"))
			})
			It("leaves RemoteAddr untouched", func() {
				call("10.0.0.1:1234", map[string]string{"X-Forwarded-For": "203.0.113.5"})
				Expect(remoteAddr).To(Equal("10.0.0.1:1234"))
			})
		})

		Context("with a trusted proxy", func() {
			BeforeEach(func() {
				conf.Server.ExtAuth.TrustedSources = "10.0.0.0/8"
			})
			It("uses the forwarded client IP when the peer is a trusted proxy", func() {
				call("10.0.0.1:1234", map[string]string{"X-Forwarded-For": "203.0.113.5, 10.0.0.1"})
				Expect(resolved).To(Equal("203.0.113.5"))
				Expect(remoteAddr).To(Equal("203.0.113.5"))
			})
			It("honours X-Real-IP from a trusted proxy", func() {
				call("10.0.0.1:1234", map[string]string{"X-Real-IP": "203.0.113.6"})
				Expect(resolved).To(Equal("203.0.113.6"))
			})
			It("ignores forwarding headers when the peer is not a trusted proxy", func() {
				call("198.51.100.9:1234", map[string]string{"X-Forwarded-For": "203.0.113.5"})
				Expect(resolved).To(Equal("198.51.100.9"))
				Expect(remoteAddr).To(Equal("198.51.100.9:1234"))
			})
			It("keeps the peer address in the context for external auth", func() {
				call("10.0.0.1:1234", map[string]string{"X-Forwarded-For": "203.0.113.5"})
				Expect(proxyIP).To(Equal("10.0.0.1:1234"))
			})
		})
	})

	Describe("ClientIPRateLimiter", func() {
		var handler http.Handler
		JustBeforeEach(func() {
			handler = realIPMiddleware(ClientIPRateLimiter(2, time.Minute)(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })))
		})
		attempt := func(peer string, header, value string) int {
			r := httptest.NewRequest("POST", "/auth/login", nil)
			r.RemoteAddr = peer
			r.Header.Set(header, value)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			return w.Code
		}

		DescribeTable("keeps one bucket per peer when the forwarding header is rotated",
			func(header string) {
				Expect(attempt("198.51.100.9:1", header, "203.0.113.1")).To(Equal(http.StatusOK))
				Expect(attempt("198.51.100.9:2", header, "203.0.113.2")).To(Equal(http.StatusOK))
				Expect(attempt("198.51.100.9:3", header, "203.0.113.3")).To(Equal(http.StatusTooManyRequests))
			},
			Entry("X-Forwarded-For", "X-Forwarded-For"),
			Entry("X-Real-IP", "X-Real-IP"),
			Entry("True-Client-IP", "True-Client-IP"),
		)

		Context("behind a trusted proxy", func() {
			BeforeEach(func() {
				conf.Server.ExtAuth.TrustedSources = "10.0.0.0/8"
			})
			It("gives each real client its own bucket", func() {
				Expect(attempt("10.0.0.1:1", "X-Forwarded-For", "203.0.113.1")).To(Equal(http.StatusOK))
				Expect(attempt("10.0.0.1:2", "X-Forwarded-For", "203.0.113.1")).To(Equal(http.StatusOK))
				Expect(attempt("10.0.0.1:3", "X-Forwarded-For", "203.0.113.1")).To(Equal(http.StatusTooManyRequests))
				Expect(attempt("10.0.0.1:4", "X-Forwarded-For", "203.0.113.2")).To(Equal(http.StatusOK))
			})
		})
	})
})
