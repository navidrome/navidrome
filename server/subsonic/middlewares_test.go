package subsonic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/deluan/navidrome/engine"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	. "github.com/onsi/ginkgo"
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

	BeforeEach(func() {
		next = &mockHandler{}
		w = httptest.NewRecorder()
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
		It("passes when all required params are available", func() {
			r := newGetRequest("u=user", "v=1.15", "c=test")
			cp := checkRequiredParameters(next)
			cp.ServeHTTP(w, r)

			Expect(next.req.Context().Value("username")).To(Equal("user"))
			Expect(next.req.Context().Value("version")).To(Equal("1.15"))
			Expect(next.req.Context().Value("client")).To(Equal("test"))
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
		var mockedUsers *mockUsers
		BeforeEach(func() {
			mockedUsers = &mockUsers{}
		})

		It("passes all parameters to users.Authenticate ", func() {
			r := newGetRequest("u=valid", "p=password", "t=token", "s=salt", "jwt=jwt")
			cp := authenticate(mockedUsers)(next)
			cp.ServeHTTP(w, r)

			Expect(mockedUsers.username).To(Equal("valid"))
			Expect(mockedUsers.password).To(Equal("password"))
			Expect(mockedUsers.token).To(Equal("token"))
			Expect(mockedUsers.salt).To(Equal("salt"))
			Expect(mockedUsers.jwt).To(Equal("jwt"))
			Expect(next.called).To(BeTrue())
			user := next.req.Context().Value("user").(model.User)
			Expect(user.UserName).To(Equal("valid"))
		})

		It("fails authentication with wrong password", func() {
			r := newGetRequest("u=invalid", "", "", "")
			cp := authenticate(mockedUsers)(next)
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
			ctx := context.WithValue(r.Context(), "username", "someone")
			ctx = context.WithValue(ctx, "client", "client")
			r = r.WithContext(ctx)
		})

		It("returns a new player in the cookies when none is specified", func() {
			gp := getPlayer(mockedPlayers)(next)
			gp.ServeHTTP(w, r)

			cookieStr := w.Header().Get("Set-Cookie")
			Expect(cookieStr).To(ContainSubstring(playerIDCookieName("someone")))
		})

		Context("PlayerId specified in Cookies", func() {
			BeforeEach(func() {
				cookie := &http.Cookie{
					Name:   playerIDCookieName("someone"),
					Value:  "123",
					MaxAge: cookieExpiry,
				}
				r.AddCookie(cookie)

				gp := getPlayer(mockedPlayers)(next)
				gp.ServeHTTP(w, r)
			})

			It("stores the player in the context", func() {
				Expect(next.called).To(BeTrue())
				player := next.req.Context().Value("player").(model.Player)
				Expect(player.ID).To(Equal("123"))
				Expect(next.req.Context().Value("transcoding")).To(BeNil())
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
					MaxAge: cookieExpiry,
				}
				r.AddCookie(cookie)
				mockedPlayers.transcoding = &model.Transcoding{ID: "12"}
				gp := getPlayer(mockedPlayers)(next)
				gp.ServeHTTP(w, r)
			})

			It("stores the player in the context", func() {
				player := next.req.Context().Value("player").(model.Player)
				Expect(player.ID).To(Equal("123"))
				transcoding := next.req.Context().Value("transcoding").(model.Transcoding)
				Expect(transcoding.ID).To(Equal("12"))
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

type mockUsers struct {
	engine.Users
	username, password, token, salt, jwt string
}

func (m *mockUsers) Authenticate(ctx context.Context, username, password, token, salt, jwt string) (*model.User, error) {
	m.username = username
	m.password = password
	m.token = token
	m.salt = salt
	m.jwt = jwt
	if username == "valid" {
		return &model.User{UserName: username, Password: password}, nil
	}
	return nil, model.ErrInvalidAuth
}

type mockPlayers struct {
	engine.Players
	transcoding *model.Transcoding
}

func (mp *mockPlayers) Get(ctx context.Context, playerId string) (*model.Player, error) {
	return &model.Player{ID: playerId}, nil
}

func (mp *mockPlayers) Register(ctx context.Context, id, client, typ, ip string) (*model.Player, *model.Transcoding, error) {
	return &model.Player{ID: id}, mp.transcoding, nil
}
