package subsonic

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func newGetRequest(queryParams ...string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/ping?"+strings.Join(queryParams, "&"), nil)
	ctx := r.Context()
	return r.WithContext(log.NewContext(ctx))
}

func newPostRequest(queryParam string, formFields ...string) *http.Request {
	GinkgoHelper()
	r, err := http.NewRequest(http.MethodPost, "/ping?"+queryParam, strings.NewReader(strings.Join(formFields, "&")))
	Expect(err).ToNot(HaveOccurred())
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
	ctx := r.Context()
	return r.WithContext(log.NewContext(ctx))
}

var _ = Describe("Middlewares", func() {
	var ctx context.Context
	var next *mockHandler
	var w *httptest.ResponseRecorder
	var ds model.DataStore

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		next = &mockHandler{}
		w = httptest.NewRecorder()
		ds = persistence.New(db.Db())
	})

	Describe("ParsePostForm", func() {
		It("converts any filed in a x-www-form-urlencoded POST into query params", func() {
			r := newPostRequest("a=abc", "u=user", "v=1.15", "c=test")
			cp := postFormToQueryParams(next)
			cp.ServeHTTP(w, r)

			Expect(next.req.URL.Query().Get("a")).To(Equal("abc"))
			Expect(next.req.URL.Query().Get("u")).To(Equal("user"))
			Expect(next.req.URL.Query().Get("v")).To(Equal("1.15"))
			Expect(next.req.URL.Query().Get("c")).To(Equal("test"))
		})
		It("adds repeated params", func() {
			r := newPostRequest("a=abc", "id=1", "id=2")
			cp := postFormToQueryParams(next)
			cp.ServeHTTP(w, r)

			Expect(next.req.URL.Query().Get("a")).To(Equal("abc"))
			Expect(next.req.URL.Query()["id"]).To(ConsistOf("1", "2"))
		})
		It("overrides query params with same key", func() {
			r := newPostRequest("a=query", "a=body")
			cp := postFormToQueryParams(next)
			cp.ServeHTTP(w, r)

			Expect(next.req.URL.Query().Get("a")).To(Equal("body"))
		})
	})

	Describe("CheckParams", func() {
		It("passes when all required params are available (subsonicauth case)", func() {
			r := newGetRequest("u=user", "v=1.15", "c=test")
			cp := checkRequiredParameters(next)
			cp.ServeHTTP(w, r)

			username, _ := request.UsernameFrom(next.req.Context())
			Expect(username).To(Equal("user"))
			version, _ := request.VersionFrom(next.req.Context())
			Expect(version).To(Equal("1.15"))
			client, _ := request.ClientFrom(next.req.Context())
			Expect(client).To(Equal("test"))

			Expect(next.called).To(BeTrue())
		})

		It("passes when all required params are available (reverse-proxy case)", func() {
			conf.Update(func(c *conf.Options) {
				c.ExtAuth.TrustedSources = "127.0.0.234/32"
				c.ExtAuth.UserHeader = "Remote-User"
			})

			r := newGetRequest("v=1.15", "c=test")
			r.Header.Add("Remote-User", "user")
			r = r.WithContext(request.WithReverseProxyIp(r.Context(), "127.0.0.234"))

			cp := checkRequiredParameters(next)
			cp.ServeHTTP(w, r)

			username, _ := request.UsernameFrom(next.req.Context())
			Expect(username).To(Equal("user"))
			version, _ := request.VersionFrom(next.req.Context())
			Expect(version).To(Equal("1.15"))
			client, _ := request.ClientFrom(next.req.Context())
			Expect(client).To(Equal("test"))

			Expect(next.called).To(BeTrue())
		})

		It("does not require u when apiKey is present", func() {
			r := newGetRequest("apiKey=nds_abc", "v=1.15", "c=test")
			cp := checkRequiredParameters(next)
			cp.ServeHTTP(w, r)

			Expect(next.called).To(BeTrue())
		})

		It("fails when user is missing", func() {
			r := newGetRequest("v=1.15", "c=test")
			cp := checkRequiredParameters(next)
			cp.ServeHTTP(w, r)

			Expect(w.Body.String()).To(ContainSubstring(`code="10"`))
			Expect(next.called).To(BeFalse())
		})

		It("fails when version is missing", func() {
			r := newGetRequest("u=user", "c=test")
			cp := checkRequiredParameters(next)
			cp.ServeHTTP(w, r)

			Expect(w.Body.String()).To(ContainSubstring(`code="10"`))
			Expect(next.called).To(BeFalse())
		})

		It("fails when client is missing", func() {
			r := newGetRequest("u=user", "v=1.15")
			cp := checkRequiredParameters(next)
			cp.ServeHTTP(w, r)

			Expect(w.Body.String()).To(ContainSubstring(`code="10"`))
			Expect(next.called).To(BeFalse())
		})
	})

	Describe("Authenticate", func() {
		BeforeEach(func() {
			ur := ds.User()
			_ = ur.Put(ctx, &model.User{
				UserName:    "admin",
				NewPassword: "wordpass",
			})
		})

		When("using password authentication", func() {
			It("passes authentication with correct credentials", func() {
				r := newGetRequest("u=admin", "p=wordpass")
				cp := authenticate(ds)(next)
				cp.ServeHTTP(w, r)

				Expect(next.called).To(BeTrue())
				user, _ := request.UserFrom(next.req.Context())
				Expect(user.UserName).To(Equal("admin"))
			})

			It("fails authentication with invalid user", func() {
				r := newGetRequest("u=invalid", "p=wordpass")
				cp := authenticate(ds)(next)
				cp.ServeHTTP(w, r)

				Expect(w.Body.String()).To(ContainSubstring(`code="40"`))
				Expect(next.called).To(BeFalse())
			})

			It("fails authentication with invalid password", func() {
				r := newGetRequest("u=admin", "p=INVALID")
				cp := authenticate(ds)(next)
				cp.ServeHTTP(w, r)

				Expect(w.Body.String()).To(ContainSubstring(`code="40"`))
				Expect(next.called).To(BeFalse())
			})
		})

		When("using token authentication", func() {
			var salt = "12345"

			It("passes authentication with correct token", func() {
				token := fmt.Sprintf("%x", md5.Sum([]byte("wordpass"+salt)))
				r := newGetRequest("u=admin", "t="+token, "s="+salt)
				cp := authenticate(ds)(next)
				cp.ServeHTTP(w, r)

				Expect(next.called).To(BeTrue())
				user, _ := request.UserFrom(next.req.Context())
				Expect(user.UserName).To(Equal("admin"))
			})

			It("fails authentication with invalid token", func() {
				r := newGetRequest("u=admin", "t=INVALID", "s="+salt)
				cp := authenticate(ds)(next)
				cp.ServeHTTP(w, r)

				Expect(w.Body.String()).To(ContainSubstring(`code="40"`))
				Expect(next.called).To(BeFalse())
			})

			It("fails authentication with empty password", func() {
				// Token generated with random Salt, empty password
				token := fmt.Sprintf("%x", md5.Sum([]byte(""+salt)))
				r := newGetRequest("u=NON_EXISTENT_USER", "t="+token, "s="+salt)
				cp := authenticate(ds)(next)
				cp.ServeHTTP(w, r)

				Expect(w.Body.String()).To(ContainSubstring(`code="40"`))
				Expect(next.called).To(BeFalse())
			})
		})

		When("using JWT authentication", func() {
			var validToken string

			BeforeEach(func() {
				conf.Update(func(c *conf.Options) { c.SessionTimeout = time.Minute })
				auth.Init(ds)
			})

			It("passes authentication with correct token", func() {
				usr := &model.User{UserName: "admin"}
				var err error
				validToken, err = auth.CreateToken(usr)
				Expect(err).NotTo(HaveOccurred())

				r := newGetRequest("u=admin", "jwt="+validToken)
				cp := authenticate(ds)(next)
				cp.ServeHTTP(w, r)

				Expect(next.called).To(BeTrue())
				user, _ := request.UserFrom(next.req.Context())
				Expect(user.UserName).To(Equal("admin"))
			})

			It("fails authentication with invalid token", func() {
				r := newGetRequest("u=admin", "jwt=INVALID_TOKEN")
				cp := authenticate(ds)(next)
				cp.ServeHTTP(w, r)

				Expect(w.Body.String()).To(ContainSubstring(`code="40"`))
				Expect(next.called).To(BeFalse())
			})
		})

		When("using reverse proxy authentication", func() {
			BeforeEach(func() {
				conf.Update(func(c *conf.Options) {
					c.ExtAuth.TrustedSources = "192.168.1.1/24"
					c.ExtAuth.UserHeader = "Remote-User"
				})
			})

			It("passes authentication with correct IP and header", func() {
				r := newGetRequest("u=admin")
				r.Header.Add("Remote-User", "admin")
				r = r.WithContext(request.WithReverseProxyIp(r.Context(), "192.168.1.1"))
				cp := authenticate(ds)(next)
				cp.ServeHTTP(w, r)

				Expect(next.called).To(BeTrue())
				user, _ := request.UserFrom(next.req.Context())
				Expect(user.UserName).To(Equal("admin"))
			})

			It("fails authentication with wrong IP", func() {
				r := newGetRequest("u=admin")
				r.Header.Add("Remote-User", "admin")
				r = r.WithContext(request.WithReverseProxyIp(r.Context(), "192.168.2.1"))
				cp := authenticate(ds)(next)
				cp.ServeHTTP(w, r)

				Expect(w.Body.String()).To(ContainSubstring(`code="40"`))
				Expect(next.called).To(BeFalse())
			})
		})

		When("using internal authentication", func() {
			It("passes authentication with correct internal credentials", func() {
				// Simulate internal authentication by setting the context with WithInternalAuth
				r := newGetRequest()
				r = r.WithContext(request.WithInternalAuth(r.Context(), "admin"))
				cp := authenticate(ds)(next)
				cp.ServeHTTP(w, r)

				Expect(next.called).To(BeTrue())
				user, _ := request.UserFrom(next.req.Context())
				Expect(user.UserName).To(Equal("admin"))
			})

			It("fails authentication with missing internal context", func() {
				r := newGetRequest("u=admin")
				// Do not set the internal auth context
				cp := authenticate(ds)(next)
				cp.ServeHTTP(w, r)

				// Internal auth requires the context, so this should fail
				Expect(w.Body.String()).To(ContainSubstring(`code="40"`))
				Expect(next.called).To(BeFalse())
			})
		})

		When("using API key authentication", func() {
			var key string
			serve := func(params ...string) {
				authenticate(ds)(next).ServeHTTP(w, newGetRequest(params...))
			}

			BeforeEach(func() {
				usr, err := ds.User().FindByUsername(ctx, "admin")
				Expect(err).ToNot(HaveOccurred())
				Expect(ds.Player().Put(ctx, &model.Player{ID: "player-1", Name: "My Phone", UserId: usr.ID, Client: "Symfonium"})).To(Succeed())
				key = "nds_0123456789abcdefghijkl"
				Expect(ds.Player().SetAPIKey(request.WithUser(ctx, *usr), "player-1", key)).To(Succeed())
			})

			It("authenticates the owner and binds the key's player", func() {
				serve("apiKey=" + key)

				Expect(next.called).To(BeTrue())
				user, _ := request.UserFrom(next.req.Context())
				Expect(user.UserName).To(Equal("admin"))
				username, _ := request.UsernameFrom(next.req.Context())
				Expect(username).To(Equal("admin"))
				player, ok := request.PlayerFrom(next.req.Context())
				Expect(ok).To(BeTrue())
				Expect(player.ID).To(Equal("player-1"))
			})

			It("accepts the key in a POST form body", func() {
				r := newPostRequest("", "apiKey="+key)
				cp := postFormToQueryParams(authenticate(ds)(next))
				cp.ServeHTTP(w, r)

				Expect(next.called).To(BeTrue())
				player, _ := request.PlayerFrom(next.req.Context())
				Expect(player.ID).To(Equal("player-1"))
			})

			It("rejects an unknown key with error 44", func() {
				serve("apiKey=nds_unknown")

				Expect(w.Body.String()).To(ContainSubstring(`code="44"`))
				Expect(next.called).To(BeFalse())
			})

			DescribeTable("rejects apiKey mixed with other credentials with error 43",
				func(extra string) {
					serve("apiKey="+key, extra)

					Expect(w.Body.String()).To(ContainSubstring(`code="43"`))
					Expect(next.called).To(BeFalse())
				},
				Entry("u", "u=admin"),
				Entry("p", "p=wordpass"),
				Entry("t", "t=abc"),
				Entry("s", "s=abc"),
				Entry("jwt", "jwt=abc"),
				Entry("empty u", "u="),
				Entry("empty p", "p="),
			)

			Context("key sent as the password", func() {
				It("authenticates and binds the key's player", func() {
					serve("u=admin", "p="+key)

					Expect(next.called).To(BeTrue())
					player, ok := request.PlayerFrom(next.req.Context())
					Expect(ok).To(BeTrue())
					Expect(player.ID).To(Equal("player-1"))
				})

				It("accepts the hex-encoded form", func() {
					serve("u=admin", "p=enc:"+hex.EncodeToString([]byte(key)))

					Expect(next.called).To(BeTrue())
				})

				It("still accepts a real password that starts with the key prefix", func() {
					Expect(ds.User().Put(ctx, &model.User{UserName: "prefixed", NewPassword: "nds_secret"})).To(Succeed())
					serve("u=prefixed", "p=nds_secret")

					Expect(next.called).To(BeTrue())
					_, ok := request.PlayerFrom(next.req.Context())
					Expect(ok).To(BeFalse())
				})

				It("rejects another user's key with error 40", func() {
					Expect(ds.User().Put(ctx, &model.User{UserName: "other", NewPassword: "pw"})).To(Succeed())
					serve("u=other", "p="+key)

					Expect(w.Body.String()).To(ContainSubstring(`code="40"`))
					Expect(next.called).To(BeFalse())
				})
			})
		})

		When("failed attempts reach AuthRequestLimit", func() {
			var cp http.Handler

			BeforeEach(func() {
				conf.Update(func(c *conf.Options) {
					c.AuthRequestLimit = 3
					c.AuthWindowLength = time.Minute
				})
				cp = authenticate(ds)(next)
			})

			serve := func(r *http.Request) *httptest.ResponseRecorder {
				next.called = false
				rec := httptest.NewRecorder()
				cp.ServeHTTP(rec, r)
				return rec
			}
			failTimes := func(n int, params ...string) {
				for range n {
					Expect(serve(newGetRequest(params...)).Body.String()).To(ContainSubstring(`code="40"`))
				}
			}

			It("rejects the correct password exactly like a wrong one", func() {
				failTimes(3, "u=admin", "p=WRONG")

				rec := serve(newGetRequest("u=admin", "p=wordpass"))

				Expect(next.called).To(BeFalse())
				Expect(rec.Code).To(Equal(http.StatusOK))
				Expect(rec.Body.String()).To(ContainSubstring(`code="40"`))
				Expect(rec.Header().Get("Retry-After")).To(BeEmpty())
			})

			It("counts attempts against unknown usernames", func() {
				failTimes(3, "u=newuser", "p=secret")
				_ = ds.User().Put(ctx, &model.User{UserName: "newuser", NewPassword: "secret"})

				serve(newGetRequest("u=newuser", "p=secret"))
				Expect(next.called).To(BeFalse())
			})

			It("treats usernames case-insensitively", func() {
				failTimes(3, "u=ADMIN", "p=WRONG")

				serve(newGetRequest("u=admin", "p=wordpass"))
				Expect(next.called).To(BeFalse())
			})

			It("does not count successful logins", func() {
				for range 10 {
					serve(newGetRequest("u=admin", "p=wordpass"))
					Expect(next.called).To(BeTrue())
				}
			})

			It("does not count server errors", func() {
				store := &failingAuthStore{}
				cp = authenticate(persistencetest.Overlay(ds, store.wrap))(next)
				store.userErr = errors.New("db down")
				failTimes(5, "u=admin", "p=wordpass")
				store.userErr = nil

				serve(newGetRequest("u=admin", "p=wordpass"))
				Expect(next.called).To(BeTrue())
			})

			It("does not count server errors when a key is sent as the password", func() {
				usr, _ := ds.User().FindByUsername(ctx, "admin")
				Expect(ds.Player().Put(ctx, &model.Player{ID: "player-1", UserId: usr.ID})).To(Succeed())
				key := "nds_0123456789abcdefghijkl"
				Expect(ds.Player().SetAPIKey(request.WithUser(ctx, *usr), "player-1", key)).To(Succeed())
				store := &failingAuthStore{}
				cp = authenticate(persistencetest.Overlay(ds, store.wrap))(next)

				store.playerErr = errors.New("db down")
				failTimes(5, "u=admin", "p="+key)
				store.playerErr = nil

				serve(newGetRequest("u=admin", "p="+key))
				Expect(next.called).To(BeTrue())
			})

			It("does not block other usernames from the same IP", func() {
				_ = ds.User().Put(ctx, &model.User{UserName: "other", NewPassword: "otherpass"})
				failTimes(3, "u=admin", "p=WRONG")

				serve(newGetRequest("u=other", "p=otherpass"))
				Expect(next.called).To(BeTrue())
			})

			It("does not block the same username from another IP", func() {
				failTimes(3, "u=admin", "p=WRONG")

				r := newGetRequest("u=admin", "p=wordpass")
				r.RemoteAddr = "198.51.100.7:1234"
				serve(r)
				Expect(next.called).To(BeTrue())
			})

			It("does not limit reverse proxy authentication", func() {
				conf.Update(func(c *conf.Options) {
					c.ExtAuth.TrustedSources = "192.168.1.1/24"
					c.ExtAuth.UserHeader = "Remote-User"
				})
				failTimes(3, "u=admin", "p=WRONG")

				r := newGetRequest()
				r.Header.Add("Remote-User", "admin")
				r = r.WithContext(request.WithReverseProxyIp(r.Context(), "192.168.1.1"))
				serve(r)
				Expect(next.called).To(BeTrue())
			})

			It("throttles a repeated bad key without locking out valid keys from the same IP", func() {
				usr, _ := ds.User().FindByUsername(ctx, "admin")
				ownerCtx := request.WithUser(ctx, *usr)
				Expect(ds.Player().Put(ctx, &model.Player{ID: "player-1", UserId: usr.ID})).To(Succeed())
				key := "nds_0123456789abcdefghijkl"
				Expect(ds.Player().SetAPIKey(ownerCtx, "player-1", key)).To(Succeed())
				badKey := "nds_bad0000000000000000000"

				for range 3 {
					Expect(serve(newGetRequest("apiKey=" + badKey)).Body.String()).To(ContainSubstring(`code="44"`))
				}
				Expect(ds.Player().Put(ctx, &model.Player{ID: "player-2", UserId: usr.ID})).To(Succeed())
				Expect(ds.Player().SetAPIKey(ownerCtx, "player-2", badKey)).To(Succeed())
				rec := serve(newGetRequest("apiKey=" + badKey))
				Expect(next.called).To(BeFalse())
				Expect(rec.Body.String()).To(ContainSubstring(`code="44"`))

				serve(newGetRequest("apiKey=" + key))
				Expect(next.called).To(BeTrue())
			})

			It("is disabled when AuthRequestLimit is 0", func() {
				conf.Update(func(c *conf.Options) { c.AuthRequestLimit = 0 })
				cp = authenticate(ds)(next)
				failTimes(10, "u=admin", "p=WRONG")

				serve(newGetRequest("u=admin", "p=wordpass"))
				Expect(next.called).To(BeTrue())
			})
		})

		When("valid requests overlap", func() {
			var gate *gatedUserRepo
			var gatedDS model.DataStore

			BeforeEach(func() {
				conf.Update(func(c *conf.Options) {
					c.AuthRequestLimit = 5
					c.AuthWindowLength = time.Minute
				})
				gate = &gatedUserRepo{
					UserRepository: ds.User(),
					entered:        make(chan struct{}, 64),
					proceed:        make(chan struct{}),
				}
				gatedDS = &gatedDataStore{DataStore: ds, users: gate}
			})

			It("lets every valid request through while checks are in flight", func() {
				const burst = 6
				cp := authenticate(gatedDS)(&countingHandler{})
				var passed atomic.Int32
				var wg sync.WaitGroup
				for range burst {
					wg.Go(func() {
						rec := httptest.NewRecorder()
						cp.ServeHTTP(rec, newGetRequest("u=admin", "p=wordpass"))
						if !strings.Contains(rec.Body.String(), `code="40"`) {
							passed.Add(1)
						}
					})
				}
				for range conf.Server().AuthRequestLimit {
					Eventually(gate.entered).Should(Receive())
				}
				close(gate.proceed)
				wg.Wait()

				Expect(passed.Load()).To(Equal(int32(burst)))
			})

			It("caps concurrent credential checks for wrong passwords", func() {
				cp := authenticate(gatedDS)(&countingHandler{})
				var wg sync.WaitGroup
				for i := range 100 {
					wg.Go(func() {
						cp.ServeHTTP(httptest.NewRecorder(), newGetRequest("u=admin", fmt.Sprintf("p=wrong%d", i)))
					})
				}

				limit := int32(conf.Server().AuthRequestLimit)
				Eventually(gate.lookups.Load).Should(Equal(limit))
				Consistently(gate.lookups.Load, 100*time.Millisecond).Should(Equal(limit))
				close(gate.proceed)
				wg.Wait()
			})
		})
	})

	Describe("AdminOnly", func() {
		It("passes admin users", func() {
			r := newGetRequest()
			r = r.WithContext(request.WithUser(r.Context(), model.User{ID: "admin-id", IsAdmin: true}))

			adminOnly(next).ServeHTTP(w, r)

			Expect(next.called).To(BeTrue())
		})

		It("rejects non-admin users", func() {
			r := newGetRequest()
			r = r.WithContext(request.WithUser(r.Context(), model.User{ID: "user-id", IsAdmin: false}))

			adminOnly(next).ServeHTTP(w, r)

			Expect(w.Body.String()).To(ContainSubstring(`code="50"`))
			Expect(next.called).To(BeFalse())
		})

		It("returns an internal error when user is missing from context", func() {
			r := newGetRequest()

			adminOnly(next).ServeHTTP(w, r)

			Expect(w.Body.String()).To(ContainSubstring(`code="0"`))
			Expect(next.called).To(BeFalse())
		})
	})

	Describe("GetPlayer", func() {
		var mockedPlayers *mockPlayers
		var r *http.Request
		BeforeEach(func() {
			mockedPlayers = &mockPlayers{}
			r = newGetRequest()
			ctx := request.WithUsername(r.Context(), "someone")
			ctx = request.WithClient(ctx, "client")
			r = r.WithContext(ctx)
		})

		It("returns a new player in the cookies when none is specified", func() {
			gp := getPlayer(mockedPlayers)(next)
			gp.ServeHTTP(w, r)

			cookieStr := w.Header().Get("Set-Cookie")
			Expect(cookieStr).To(ContainSubstring(playerIDCookieName("someone")))
		})

		It("registers the player with the client IP from RemoteAddr", func() {
			r.RemoteAddr = "192.0.2.10:54321"
			gp := middleware.ClientIPFromRemoteAddr(getPlayer(mockedPlayers)(next))
			gp.ServeHTTP(w, r)

			Expect(mockedPlayers.ip).To(Equal("192.0.2.10"))
		})

		It("keeps the client IP when RemoteAddr has no port, as set behind a trusted proxy", func() {
			r.RemoteAddr = "192.0.2.10"
			gp := middleware.ClientIPFromRemoteAddr(getPlayer(mockedPlayers)(next))
			gp.ServeHTTP(w, r)

			Expect(mockedPlayers.ip).To(Equal("192.0.2.10"))
		})

		It("takes the client IP resolved into the context, not the peer address", func() {
			// realIPMiddleware resolves forwarding headers from a trusted proxy the same way
			r.RemoteAddr = "10.0.0.1:1234"
			r.Header.Set("X-Real-IP", "203.0.113.6")
			gp := middleware.ClientIPFromHeader("X-Real-IP")(getPlayer(mockedPlayers)(next))
			gp.ServeHTTP(w, r)

			Expect(mockedPlayers.ip).To(Equal("203.0.113.6"))
		})

		It("does not take a unix socket peer for an IP", func() {
			r.RemoteAddr = "@"
			gp := middleware.ClientIPFromRemoteAddr(getPlayer(mockedPlayers)(next))
			gp.ServeHTTP(w, r)

			Expect(mockedPlayers.ip).To(BeEmpty())
		})

		It("does not add the cookie if there was an error", func() {
			ctx := request.WithClient(r.Context(), "error")
			r = r.WithContext(ctx)

			gp := getPlayer(mockedPlayers)(next)
			gp.ServeHTTP(w, r)

			cookieStr := w.Header().Get("Set-Cookie")
			Expect(cookieStr).To(BeEmpty())
		})

		Context("player bound by an API key", func() {
			BeforeEach(func() {
				r = r.WithContext(request.WithPlayer(r.Context(), model.Player{ID: "keyed"}))
				gp := getPlayer(mockedPlayers)(next)
				gp.ServeHTTP(w, r)
			})

			It("uses the key's player", func() {
				Expect(mockedPlayers.touched).To(BeTrue())
				player, _ := request.PlayerFrom(next.req.Context())
				Expect(player.ID).To(Equal("keyed"))
			})

			It("does not set the player cookie", func() {
				Expect(w.Header().Get("Set-Cookie")).To(BeEmpty())
			})
		})

		Context("PlayerId specified in Cookies", func() {
			BeforeEach(func() {
				cookie := &http.Cookie{
					Name:   playerIDCookieName("someone"),
					Value:  "123",
					MaxAge: consts.CookieExpiry,
				}
				r.AddCookie(cookie)

				gp := getPlayer(mockedPlayers)(next)
				gp.ServeHTTP(w, r)
			})

			It("stores the player in the context", func() {
				Expect(next.called).To(BeTrue())
				player, _ := request.PlayerFrom(next.req.Context())
				Expect(player.ID).To(Equal("123"))
				_, ok := request.TranscodingFrom(next.req.Context())
				Expect(ok).To(BeFalse())
			})

			It("returns the playerId in the cookie", func() {
				cookieStr := w.Header().Get("Set-Cookie")
				Expect(cookieStr).To(ContainSubstring(playerIDCookieName("someone") + "=123"))
			})
		})

		Context("Player has transcoding configured", func() {
			BeforeEach(func() {
				cookie := &http.Cookie{
					Name:   playerIDCookieName("someone"),
					Value:  "123",
					MaxAge: consts.CookieExpiry,
				}
				r.AddCookie(cookie)
				mockedPlayers.transcoding = &model.Transcoding{ID: "12"}
				gp := getPlayer(mockedPlayers)(next)
				gp.ServeHTTP(w, r)
			})

			It("stores the player in the context", func() {
				player, _ := request.PlayerFrom(next.req.Context())
				Expect(player.ID).To(Equal("123"))
				transcoding, _ := request.TranscodingFrom(next.req.Context())
				Expect(transcoding.ID).To(Equal("12"))
			})
		})
	})

	Describe("validateCredentials", func() {
		var usr *model.User

		BeforeEach(func() {
			ur := ds.User()
			_ = ur.Put(ctx, &model.User{
				UserName:    "admin",
				NewPassword: "wordpass",
			})

			var err error
			usr, err = ur.FindByUsernameWithPassword(ctx, "admin")
			Expect(err).ToNot(HaveOccurred())
		})

		Context("Plaintext password", func() {
			It("authenticates with plaintext password ", func() {
				err := validateCredentials(usr, "wordpass", "", "", "")
				Expect(err).NotTo(HaveOccurred())
			})

			It("fails authentication with wrong password", func() {
				err := validateCredentials(usr, "INVALID", "", "", "")
				Expect(err).To(MatchError(model.ErrInvalidAuth))
			})
		})

		Context("Encoded password", func() {
			It("authenticates with simple encoded password ", func() {
				err := validateCredentials(usr, "enc:776f726470617373", "", "", "")
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("Token based authentication", func() {
			It("authenticates with token based authentication", func() {
				err := validateCredentials(usr, "", "23b342970e25c7928831c3317edd0b67", "retnlmjetrymazgkt", "")
				Expect(err).NotTo(HaveOccurred())
			})

			It("fails if salt is missing", func() {
				err := validateCredentials(usr, "", "23b342970e25c7928831c3317edd0b67", "", "")
				Expect(err).To(MatchError(model.ErrInvalidAuth))
			})
		})

		Context("JWT based authentication", func() {
			var usr *model.User
			var validToken string

			BeforeEach(func() {
				conf.Update(func(c *conf.Options) { c.SessionTimeout = time.Minute })
				auth.Init(ds)

				usr = &model.User{UserName: "admin"}
				var err error
				validToken, err = auth.CreateToken(usr)
				Expect(err).ToNot(HaveOccurred())
			})

			It("authenticates with JWT token based authentication", func() {
				err := validateCredentials(usr, "", "", "", validToken)

				Expect(err).NotTo(HaveOccurred())
			})

			It("fails if JWT token is invalid", func() {
				err := validateCredentials(usr, "", "", "", "invalid.token")
				Expect(err).To(MatchError(model.ErrInvalidAuth))
			})

			It("fails if JWT token sub is different than username", func() {
				u := &model.User{UserName: "hacker"}
				validToken, _ = auth.CreateToken(u)
				err := validateCredentials(usr, "", "", "", validToken)
				Expect(err).To(MatchError(model.ErrInvalidAuth))
			})
		})

		Context("JWT credentials", func() {
			var usr *model.User

			BeforeEach(func() {
				conf.Update(func(c *conf.Options) { c.SessionTimeout = time.Minute })
				auth.Init(ds)
				usr = &model.User{ID: "u1", UserName: "johndoe", TokenEpoch: 1}
			})

			It("accepts an unscoped session token", func() {
				tokenStr, err := auth.CreateToken(usr)
				Expect(err).ToNot(HaveOccurred())
				Expect(validateCredentials(usr, "", "", "", tokenStr)).To(Succeed())
			})

			It("rejects a jellyfin-scoped token", func() {
				tokenStr, err := auth.CreateAPIToken(usr, auth.AudienceJellyfin)
				Expect(err).ToNot(HaveOccurred())
				Expect(validateCredentials(usr, "", "", "", tokenStr)).To(MatchError(model.ErrInvalidAuth))
			})

			It("rejects a token with a stale epoch", func() {
				tokenStr, err := auth.CreateToken(usr)
				Expect(err).ToNot(HaveOccurred())
				usr.TokenEpoch = 2
				Expect(validateCredentials(usr, "", "", "", tokenStr)).To(MatchError(model.ErrInvalidAuth))
			})
		})
	})
})

type mockHandler struct {
	req    *http.Request
	called bool
}

func (mh *mockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mh.req = r
	mh.called = true
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

type mockPlayers struct {
	core.Players
	transcoding *model.Transcoding
	touched     bool
	ip          string
}

func (mp *mockPlayers) Touch(_ context.Context, plr model.Player, _, _, ip string) (*model.Player, *model.Transcoding, error) {
	mp.touched = true
	mp.ip = ip
	return &plr, mp.transcoding, nil
}

func (mp *mockPlayers) Get(ctx context.Context, playerId string) (*model.Player, error) {
	return &model.Player{ID: playerId}, nil
}

func (mp *mockPlayers) Register(ctx context.Context, id, client, typ, ip string) (*model.Player, *model.Transcoding, error) {
	mp.ip = ip
	if client == "error" {
		return nil, nil, errors.New(client)
	}
	return &model.Player{ID: id}, mp.transcoding, nil
}

type gatedDataStore struct {
	model.DataStore
	users model.UserRepository
}

func (g *gatedDataStore) User() model.UserRepository { return g.users }

type gatedUserRepo struct {
	model.UserRepository
	entered chan struct{}
	proceed chan struct{}
	lookups atomic.Int32
}

func (g *gatedUserRepo) FindByUsernameWithPassword(ctx context.Context, username string) (*model.User, error) {
	g.lookups.Add(1)
	g.entered <- struct{}{}
	<-g.proceed
	return g.UserRepository.FindByUsernameWithPassword(ctx, username)
}

type countingHandler struct{ calls atomic.Int32 }

func (c *countingHandler) ServeHTTP(http.ResponseWriter, *http.Request) { c.calls.Add(1) }

// failingAuthStore fails the credential lookups while userErr or playerErr is set.
type failingAuthStore struct {
	userErr, playerErr error
}

func (f *failingAuthStore) wrap(ds model.DataStore) model.DataStore {
	return failingAuthDataStore{DataStore: ds, f: f}
}

type failingAuthDataStore struct {
	model.DataStore
	f *failingAuthStore
}

func (d failingAuthDataStore) User() model.UserRepository {
	return failingAuthUserRepo{UserRepository: d.DataStore.User(), f: d.f}
}

func (d failingAuthDataStore) Player() model.PlayerRepository {
	return failingAuthPlayerRepo{PlayerRepository: d.DataStore.Player(), f: d.f}
}

type failingAuthUserRepo struct {
	model.UserRepository
	f *failingAuthStore
}

func (r failingAuthUserRepo) FindByUsernameWithPassword(ctx context.Context, username string) (*model.User, error) {
	if r.f.userErr != nil {
		return nil, r.f.userErr
	}
	return r.UserRepository.FindByUsernameWithPassword(ctx, username)
}

type failingAuthPlayerRepo struct {
	model.PlayerRepository
	f *failingAuthStore
}

func (r failingAuthPlayerRepo) FindByAPIKey(ctx context.Context, key string) (*model.Player, error) {
	if r.f.playerErr != nil {
		return nil, r.f.playerErr
	}
	return r.PlayerRepository.FindByAPIKey(ctx, key)
}
