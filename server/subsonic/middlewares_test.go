package subsonic

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func newGetRequest(queryParams ...string) *http.Request {
	r := httptest.NewRequest("GET", "/ping?"+strings.Join(queryParams, "&"), nil)
	ctx := r.Context()
	return r.WithContext(log.NewContext(ctx))
}

func newPostRequest(queryParam string, formFields ...string) *http.Request {
	r, err := http.NewRequest("POST", "/ping?"+queryParam, strings.NewReader(strings.Join(formFields, "&")))
	if err != nil {
		panic(err)
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
	ctx := r.Context()
	return r.WithContext(log.NewContext(ctx))
}

var _ = Describe("Middlewares", func() {
	var next *mockHandler
	var w *httptest.ResponseRecorder
	var ds model.DataStore

	BeforeEach(func() {
		next = &mockHandler{}
		w = httptest.NewRecorder()
		ds = &tests.MockDataStore{}
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
			conf.Server.ExtAuth.TrustedSources = "127.0.0.234/32"
			conf.Server.ExtAuth.UserHeader = "Remote-User"

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
			ur := ds.User(context.TODO())
			_ = ur.Put(&model.User{
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
				DeferCleanup(configtest.SetupConfig())
				conf.Server.SessionTimeout = time.Minute
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
				DeferCleanup(configtest.SetupConfig())
				conf.Server.ExtAuth.TrustedSources = "192.168.1.1/24"
				conf.Server.ExtAuth.UserHeader = "Remote-User"
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

		When("failed attempts reach AuthRequestLimit", func() {
			var cp http.Handler

			BeforeEach(func() {
				DeferCleanup(configtest.SetupConfig())
				conf.Server.AuthRequestLimit = 3
				conf.Server.AuthWindowLength = time.Minute
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
				_ = ds.User(context.TODO()).Put(&model.User{UserName: "newuser", NewPassword: "secret"})

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
				userRepo := ds.User(context.TODO()).(*tests.MockedUserRepo)
				userRepo.Error = errors.New("db down")
				failTimes(5, "u=admin", "p=wordpass")
				userRepo.Error = nil

				serve(newGetRequest("u=admin", "p=wordpass"))
				Expect(next.called).To(BeTrue())
			})

			It("does not block other usernames from the same IP", func() {
				_ = ds.User(context.TODO()).Put(&model.User{UserName: "other", NewPassword: "otherpass"})
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
				conf.Server.ExtAuth.TrustedSources = "192.168.1.1/24"
				conf.Server.ExtAuth.UserHeader = "Remote-User"
				failTimes(3, "u=admin", "p=WRONG")

				r := newGetRequest()
				r.Header.Add("Remote-User", "admin")
				r = r.WithContext(request.WithReverseProxyIp(r.Context(), "192.168.1.1"))
				serve(r)
				Expect(next.called).To(BeTrue())
			})

			It("is disabled when AuthRequestLimit is 0", func() {
				conf.Server.AuthRequestLimit = 0
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
				DeferCleanup(configtest.SetupConfig())
				conf.Server.AuthRequestLimit = 5
				conf.Server.AuthWindowLength = time.Minute
				gate = &gatedUserRepo{
					UserRepository: ds.User(context.TODO()),
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
				for range conf.Server.AuthRequestLimit {
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

				limit := int32(conf.Server.AuthRequestLimit)
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

		It("does not add the cookie if there was an error", func() {
			ctx := request.WithClient(r.Context(), "error")
			r = r.WithContext(ctx)

			gp := getPlayer(mockedPlayers)(next)
			gp.ServeHTTP(w, r)

			cookieStr := w.Header().Get("Set-Cookie")
			Expect(cookieStr).To(BeEmpty())
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
			ur := ds.User(context.TODO())
			_ = ur.Put(&model.User{
				UserName:    "admin",
				NewPassword: "wordpass",
			})

			var err error
			usr, err = ur.FindByUsernameWithPassword("admin")
			if err != nil {
				panic(err)
			}
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
				DeferCleanup(configtest.SetupConfig())
				conf.Server.SessionTimeout = time.Minute
				auth.Init(ds)

				usr = &model.User{UserName: "admin"}
				var err error
				validToken, err = auth.CreateToken(usr)
				if err != nil {
					panic(err)
				}
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
				DeferCleanup(configtest.SetupConfig())
				conf.Server.SessionTimeout = time.Minute
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
}

func (mp *mockPlayers) Get(ctx context.Context, playerId string) (*model.Player, error) {
	return &model.Player{ID: playerId}, nil
}

func (mp *mockPlayers) Register(ctx context.Context, id, client, typ, ip string) (*model.Player, *model.Transcoding, error) {
	if client == "error" {
		return nil, nil, errors.New(client)
	}
	return &model.Player{ID: id}, mp.transcoding, nil
}

type gatedDataStore struct {
	model.DataStore
	users model.UserRepository
}

func (g *gatedDataStore) User(context.Context) model.UserRepository { return g.users }

type gatedUserRepo struct {
	model.UserRepository
	entered chan struct{}
	proceed chan struct{}
	lookups atomic.Int32
}

func (g *gatedUserRepo) FindByUsernameWithPassword(username string) (*model.User, error) {
	g.lookups.Add(1)
	g.entered <- struct{}{}
	<-g.proceed
	return g.UserRepository.FindByUsernameWithPassword(username)
}

type countingHandler struct{ calls atomic.Int32 }

func (c *countingHandler) ServeHTTP(http.ResponseWriter, *http.Request) { c.calls.Add(1) }
