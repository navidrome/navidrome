package lastfm

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("auth_router", func() {
	var (
		ds         *tests.MockDataStore
		userProps  *tests.MockedUserPropsRepo
		httpClient *tests.FakeHttpClient
		router     *Router
	)

	const (
		victimID   = "victim-user-id"
		attackerID = "attacker-user-id"
	)

	BeforeEach(func() {
		userProps = &tests.MockedUserPropsRepo{}
		ds = &tests.MockDataStore{
			MockedProperty:  &tests.MockedPropertyRepo{},
			MockedUserProps: userProps,
		}
		auth.Init(ds)

		httpClient = &tests.FakeHttpClient{}
		router = &Router{
			ds:          ds,
			apiKey:      "API_KEY",
			secret:      "SECRET",
			sessionKeys: &agents.SessionKeys{DataStore: ds, KeyName: sessionKeyProperty},
		}
		router.client = newClient(router.apiKey, router.secret, httpClient)
		router.Handler = router.routes()
	})

	storedSessionKey := func(userID string) string {
		key, _ := userProps.Get(userID, sessionKeyProperty)
		return key
	}

	stubGetSessionOK := func(sessionKey string) {
		httpClient.Res = http.Response{
			Body:       io.NopCloser(bytes.NewBufferString(`{"session":{"name":"Navidrome","key":"` + sessionKey + `","subscriber":0}}`)),
			StatusCode: 200,
		}
	}

	Describe("getLinkStatus", func() {
		It("includes a signed linkToken for the authenticated user", func() {
			req := httptest.NewRequest(http.MethodGet, "/link", nil)
			ctx := request.WithUser(req.Context(), model.User{ID: victimID})
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			router.getLinkStatus(rec, req)

			Expect(rec.Code).To(Equal(http.StatusOK))
			var body map[string]any
			Expect(json.Unmarshal(rec.Body.Bytes(), &body)).To(Succeed())
			Expect(body["apiKey"]).To(Equal("API_KEY"))
			Expect(body["status"]).To(Equal(false))
			token, ok := body["linkToken"].(string)
			Expect(ok).To(BeTrue())
			Expect(token).ToNot(BeEmpty())

			verified, err := verifyLinkToken(token)
			Expect(err).ToNot(HaveOccurred())
			Expect(verified).To(Equal(victimID))
		})
	})

	Describe("callback", func() {
		It("stores the session key under the user encoded in the signed token", func() {
			stubGetSessionOK("LEGIT_SESSION")
			linkToken, err := createLinkToken(victimID)
			Expect(err).ToNot(HaveOccurred())

			req := httptest.NewRequest(http.MethodGet, "/link/callback?uid="+linkToken+"&token=LASTFM_TOKEN", nil)
			rec := httptest.NewRecorder()
			router.callback(rec, req)

			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(storedSessionKey(victimID)).To(Equal("LEGIT_SESSION"))
		})

		It("rejects a raw (unsigned) uid value", func() {
			req := httptest.NewRequest(http.MethodGet, "/link/callback?uid="+victimID+"&token=LASTFM_TOKEN", nil)
			rec := httptest.NewRecorder()
			router.callback(rec, req)

			Expect(rec.Code).To(Equal(http.StatusBadRequest))
			Expect(storedSessionKey(victimID)).To(BeEmpty())
			Expect(httpClient.SavedRequest).To(BeNil())
		})

		It("rejects an expired link token", func() {
			expiredToken, err := auth.EncodeToken(map[string]any{
				"uid":   victimID,
				"scope": linkTokenScope,
				"exp":   time.Now().Add(-1 * time.Minute).UTC().Unix(),
			})
			Expect(err).ToNot(HaveOccurred())

			req := httptest.NewRequest(http.MethodGet, "/link/callback?uid="+expiredToken+"&token=LASTFM_TOKEN", nil)
			rec := httptest.NewRecorder()
			router.callback(rec, req)

			Expect(rec.Code).To(Equal(http.StatusBadRequest))
			Expect(storedSessionKey(victimID)).To(BeEmpty())
			Expect(httpClient.SavedRequest).To(BeNil())
		})

		It("rejects a token with the wrong scope (e.g. a regular session JWT)", func() {
			sessionJWT, err := auth.CreateToken(&model.User{ID: attackerID, UserName: "attacker"})
			Expect(err).ToNot(HaveOccurred())

			req := httptest.NewRequest(http.MethodGet, "/link/callback?uid="+sessionJWT+"&token=LASTFM_TOKEN", nil)
			rec := httptest.NewRecorder()
			router.callback(rec, req)

			Expect(rec.Code).To(Equal(http.StatusBadRequest))
			Expect(storedSessionKey(attackerID)).To(BeEmpty())
			Expect(httpClient.SavedRequest).To(BeNil())
		})

		It("writes only under the user encoded in the token, regardless of query manipulation", func() {
			// An attacker holds a legitimate link token for their own account.
			// They attempt to call the callback hoping to overwrite the victim's
			// session key — but the handler must derive the user ID from the
			// signed token, not from any other input.
			stubGetSessionOK("ATTACKER_SESSION")
			attackerToken, err := createLinkToken(attackerID)
			Expect(err).ToNot(HaveOccurred())

			req := httptest.NewRequest(http.MethodGet, "/link/callback?uid="+attackerToken+"&token=LASTFM_TOKEN&user="+victimID, nil)
			rec := httptest.NewRecorder()
			router.callback(rec, req)

			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(storedSessionKey(attackerID)).To(Equal("ATTACKER_SESSION"))
			Expect(storedSessionKey(victimID)).To(BeEmpty())
		})

		It("returns 400 when uid is missing", func() {
			req := httptest.NewRequest(http.MethodGet, "/link/callback?token=LASTFM_TOKEN", nil)
			rec := httptest.NewRecorder()
			router.callback(rec, req)

			Expect(rec.Code).To(Equal(http.StatusBadRequest))
		})

		It("returns 400 when token is missing", func() {
			linkToken, err := createLinkToken(victimID)
			Expect(err).ToNot(HaveOccurred())

			req := httptest.NewRequest(http.MethodGet, "/link/callback?uid="+linkToken, nil)
			rec := httptest.NewRecorder()
			router.callback(rec, req)

			Expect(rec.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("link token helpers", func() {
		It("round-trips a freshly issued token", func() {
			token, err := createLinkToken(victimID)
			Expect(err).ToNot(HaveOccurred())

			uid, err := verifyLinkToken(token)
			Expect(err).ToNot(HaveOccurred())
			Expect(uid).To(Equal(victimID))
		})

		It("rejects garbage", func() {
			_, err := verifyLinkToken("not-a-jwt")
			Expect(err).To(HaveOccurred())
		})

		It("rejects a token whose scope claim is wrong", func() {
			wrongScopeToken, err := auth.EncodeToken(map[string]any{
				"uid":   victimID,
				"scope": "some-other-scope",
				"exp":   time.Now().Add(linkTokenTTL).UTC().Unix(),
			})
			Expect(err).ToNot(HaveOccurred())

			_, err = verifyLinkToken(wrongScopeToken)
			Expect(err).To(MatchError("invalid link token scope"))
		})

		It("rejects a scoped token that has no expiration", func() {
			nonExpiringToken, err := auth.EncodeToken(map[string]any{
				"uid":   victimID,
				"scope": linkTokenScope,
			})
			Expect(err).ToNot(HaveOccurred())

			_, err = verifyLinkToken(nonExpiringToken)
			Expect(err).To(MatchError("link token missing expiration"))
		})
	})
})
