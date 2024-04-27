package subsonic

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
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
			conf.Server.ReverseProxyWhitelist = "127.0.0.234/32"
			conf.Server.ReverseProxyUserHeader = "Remote-User"

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

		It("passes authentication with correct credentials", func() {
			r := newGetRequest("u=admin", "p=wordpass")
			cp := authenticate(ds)(next)
			cp.ServeHTTP(w, r)

			Expect(next.called).To(BeTrue())
			user, _ := request.UserFrom(next.req.Context())
			Expect(user.UserName).To(Equal("admin"))
		})

		It("fails authentication with wrong password", func() {
			r := newGetRequest("u=invalid", "", "", "")
			cp := authenticate(ds)(next)
			cp.ServeHTTP(w, r)

			Expect(w.Body.String()).To(ContainSubstring(`code="40"`))
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
	})
})

type mockHandler struct {
	req    *http.Request
	called bool
}

func (mh *mockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mh.req = r
	mh.called = true
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
