package jellyfin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/quickconnect"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type fullQuickConnect struct{ quickconnect.QuickConnect }

func (fullQuickConnect) Initiate(quickconnect.Device) (quickconnect.Request, error) {
	return quickconnect.Request{}, quickconnect.ErrTooManyRequests
}

var _ = Describe("QuickConnect", func() {
	const finamp = `MediaBrowser Client="Finamp", Device="Pixel 7", DeviceId="dev-1", Version="1.0.0"`
	var (
		api   *Router
		ds    *tests.MockDataStore
		qc    quickconnect.QuickConnect
		alice = model.User{ID: testID("alice"), UserName: "alice"}
		bob   = model.User{ID: testID("bob"), UserName: "bob"}
		admin = model.User{ID: testID("admin"), UserName: "admin", IsAdmin: true}
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Jellyfin.QuickConnect = true
		ds = &tests.MockDataStore{}
		auth.Init(ds)
		ur := ds.User(context.Background()).(*tests.MockedUserRepo)
		for _, u := range []model.User{alice, bob, admin} {
			Expect(ur.Put(&u)).To(Succeed())
		}
		qc = quickconnect.New()
		api = &Router{ds: ds, quickConnect: qc}
	})

	as := func(r *http.Request, u model.User) *http.Request {
		return r.WithContext(request.WithUser(r.Context(), u))
	}
	initiate := func() quickconnect.Request {
		req, err := qc.Initiate(quickconnect.Device{ID: "dev-1", Name: "Pixel 7", App: "Finamp", AppVersion: "1.0.0"})
		Expect(err).ToNot(HaveOccurred())
		return req
	}

	Describe("GET /QuickConnect/Enabled", func() {
		DescribeTable("reports the config value",
			func(enabled bool, expected string) {
				conf.Server.Jellyfin.QuickConnect = enabled
				w := httptest.NewRecorder()
				api.quickConnectEnabled(w, httptest.NewRequest("GET", "/QuickConnect/Enabled", nil))
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(w.Body.String()).To(MatchJSON(expected))
			},
			Entry("enabled", true, "true"),
			Entry("disabled", false, "false"),
		)
	})

	Describe("requireQuickConnect", func() {
		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })

		It("rejects requests with 401 when Quick Connect is disabled", func() {
			conf.Server.Jellyfin.QuickConnect = false
			w := httptest.NewRecorder()
			requireQuickConnect(next).ServeHTTP(w, httptest.NewRequest("POST", "/QuickConnect/Initiate", nil))
			Expect(w.Code).To(Equal(http.StatusUnauthorized))
		})

		It("passes requests through when Quick Connect is enabled", func() {
			w := httptest.NewRecorder()
			requireQuickConnect(next).ServeHTTP(w, httptest.NewRequest("POST", "/QuickConnect/Initiate", nil))
			Expect(w.Code).To(Equal(http.StatusTeapot))
		})
	})

	Describe("POST /QuickConnect/Initiate", func() {
		initiateWith := func(authHeader string) *httptest.ResponseRecorder {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/QuickConnect/Initiate", nil)
			r.Header.Set("Authorization", authHeader)
			api.quickConnectInitiate(w, r)
			return w
		}

		It("returns all QuickConnectResult fields, with the device from the auth header", func() {
			w := initiateWith(finamp)

			Expect(w.Code).To(Equal(http.StatusOK))
			var raw map[string]any
			Expect(json.Unmarshal(w.Body.Bytes(), &raw)).To(Succeed())
			// sdk-kotlin declares every field non-null.
			Expect(raw).To(HaveKeyWithValue("Authenticated", false))
			Expect(raw).To(HaveKeyWithValue("Secret", MatchRegexp(`^[0-9a-f]{64}$`)))
			Expect(raw).To(HaveKeyWithValue("Code", MatchRegexp(`^\d{6}$`)))
			Expect(raw).To(HaveKeyWithValue("DeviceId", "dev-1"))
			Expect(raw).To(HaveKeyWithValue("DeviceName", "Pixel 7"))
			Expect(raw).To(HaveKeyWithValue("AppName", "Finamp"))
			Expect(raw).To(HaveKeyWithValue("AppVersion", "1.0.0"))
			Expect(raw).To(HaveKeyWithValue("DateAdded", Not(BeEmpty())))

			_, err := qc.Status(raw["Secret"].(string))
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns 400 when the auth header does not identify the client", func() {
			w := initiateWith(`MediaBrowser Client="Finamp", Device="Pixel 7", Version="1.0.0"`)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("returns 400 when a client field is oversized", func() {
			long := strings.Repeat("x", maxQuickConnectField+1)
			api.quickConnect = fullQuickConnect{qc}
			w := initiateWith(`MediaBrowser Client="Finamp", Device="` + long + `", DeviceId="dev-1", Version="1.0.0"`)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("returns 429 when too many requests are pending", func() {
			api.quickConnect = fullQuickConnect{qc}
			Expect(initiateWith(finamp).Code).To(Equal(http.StatusTooManyRequests))
		})
	})

	Describe("GET /QuickConnect/Connect", func() {
		connect := func(secret string) (int, dto.QuickConnectResult) {
			w := httptest.NewRecorder()
			invoke(api.quickConnectConnect, w, httptest.NewRequest("GET", "/QuickConnect/Connect?Secret="+secret, nil))
			var res dto.QuickConnectResult
			if w.Code == http.StatusOK {
				Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			}
			return w.Code, res
		}

		It("reports whether the request has been authorized", func() {
			req := initiate()
			code, res := connect(req.Secret)
			Expect(code).To(Equal(http.StatusOK))
			Expect(res.Authenticated).To(BeFalse())
			Expect(res.Code).To(Equal(req.Code))

			_, err := qc.Authorize(req.Code, alice.ID)
			Expect(err).ToNot(HaveOccurred())
			code, res = connect(req.Secret)
			Expect(code).To(Equal(http.StatusOK))
			Expect(res.Authenticated).To(BeTrue())
		})

		It("returns 404 for an unknown secret", func() {
			code, _ := connect("unknown")
			Expect(code).To(Equal(http.StatusNotFound))
		})
	})

	Describe("POST /QuickConnect/Authorize", func() {
		authorize := func(query string, u model.User) *httptest.ResponseRecorder {
			w := httptest.NewRecorder()
			invoke(api.quickConnectAuthorize, w, as(httptest.NewRequest("POST", "/QuickConnect/Authorize?"+query, nil), u))
			return w
		}
		approver := func(req quickconnect.Request) string {
			got, err := qc.Status(req.Secret)
			Expect(err).ToNot(HaveOccurred())
			return got.UserID
		}

		It("approves the code for the caller when no userId is given", func() {
			req := initiate()
			w := authorize("code="+req.Code, alice)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(MatchJSON("true"))
			Expect(approver(req)).To(Equal(alice.ID))
		})

		It("accepts the caller's own userId", func() {
			req := initiate()
			w := authorize("Code="+req.Code+"&UserId="+dto.EncodeID(alice.ID), alice)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(approver(req)).To(Equal(alice.ID))
		})

		It("forbids a non-admin from approving for another user", func() {
			req := initiate()
			w := authorize("code="+req.Code+"&userId="+dto.EncodeID(bob.ID), alice)
			Expect(w.Code).To(Equal(http.StatusForbidden))
			Expect(approver(req)).To(BeEmpty())
		})

		It("lets an admin approve for another user", func() {
			req := initiate()
			w := authorize("code="+req.Code+"&userId="+dto.EncodeID(bob.ID), admin)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(approver(req)).To(Equal(bob.ID))
		})

		It("returns 404 when an admin approves for an unknown user", func() {
			req := initiate()
			w := authorize("code="+req.Code+"&userId="+dto.EncodeID(testID("ghost")), admin)
			Expect(w.Code).To(Equal(http.StatusNotFound))
			Expect(approver(req)).To(BeEmpty())
		})

		It("returns 400 for a malformed userId", func() {
			req := initiate()
			w := authorize("code="+req.Code+"&userId=not-a-guid", admin)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("returns 404 for an unknown code", func() {
			w := authorize("code=000000", alice)
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})

		It("returns 409 for a code that is already approved", func() {
			req := initiate()
			Expect(authorize("code="+req.Code, alice).Code).To(Equal(http.StatusOK))
			Expect(authorize("code="+req.Code, bob).Code).To(Equal(http.StatusConflict))
			Expect(approver(req)).To(Equal(alice.ID))
		})
	})

	Describe("POST /Users/AuthenticateWithQuickConnect", func() {
		redeem := func(body string) *httptest.ResponseRecorder {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/AuthenticateWithQuickConnect", strings.NewReader(body))
			r.Header.Set("Authorization", finamp)
			api.authenticateWithQuickConnect(w, r)
			return w
		}
		redeemSecret := func(secret string) *httptest.ResponseRecorder {
			return redeem(`{"Secret":"` + secret + `"}`)
		}

		It("signs in the approving user once", func() {
			req := initiate()
			_, _ = qc.Authorize(req.Code, alice.ID)

			w := redeemSecret(req.Secret)
			Expect(w.Code).To(Equal(http.StatusOK))
			var res dto.AuthenticationResult
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.User.Name).To(Equal("alice"))
			Expect(res.ServerId).ToNot(BeEmpty())
			Expect(res.SessionInfo).ToNot(BeNil())
			Expect(res.SessionInfo.DeviceId).To(Equal("dev-1"))
			claims, err := auth.Validate(res.AccessToken)
			Expect(err).ToNot(HaveOccurred())
			Expect(claims.Subject).To(Equal("alice"))

			Expect(redeemSecret(req.Secret).Code).To(Equal(http.StatusNotFound))
		})

		It("accepts a camelCase body", func() {
			req := initiate()
			_, _ = qc.Authorize(req.Code, alice.ID)
			Expect(redeem(`{"secret":"` + req.Secret + `"}`).Code).To(Equal(http.StatusOK))
		})

		It("returns 404 while the request is not approved", func() {
			req := initiate()
			Expect(redeemSecret(req.Secret).Code).To(Equal(http.StatusNotFound))
		})

		DescribeTable("returns 400 for a bad body",
			func(body string) {
				Expect(redeem(body).Code).To(Equal(http.StatusBadRequest))
			},
			Entry("not JSON", `nope`),
			Entry("no secret", `{}`),
		)

		It("returns 401 when the approving user no longer exists", func() {
			req := initiate()
			_, _ = qc.Authorize(req.Code, testID("ghost"))
			Expect(redeemSecret(req.Secret).Code).To(Equal(http.StatusUnauthorized))
		})

		It("returns 500 when the user lookup fails", func() {
			req := initiate()
			_, _ = qc.Authorize(req.Code, alice.ID)
			ds.User(context.Background()).(*tests.MockedUserRepo).Error = errors.New("db down")
			Expect(redeemSecret(req.Secret).Code).To(Equal(http.StatusInternalServerError))
		})
	})
})
