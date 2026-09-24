package e2e

import (
	"net/http/httptest"
	"net/url"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("API key authentication", func() {
	var key string

	BeforeEach(func() {
		setupTestDB()
		userCtx := request.WithUser(ctx, regularUser)
		player := &model.Player{ID: "apikey-player", Name: "Phone", UserId: regularUser.ID, Client: "test-client"}
		Expect(ds.Player(userCtx).Put(player)).To(Succeed())
		key = "nav_0123456789abcdefghijkl"
		Expect(ds.Player(userCtx).SetAPIKey(player.ID, key)).To(Succeed())
	})

	doKeyReq := func(endpoint, apiKey string) *responses.Subsonic {
		q := url.Values{"apiKey": {apiKey}, "v": {"1.16.1"}, "c": {"test-client"}, "f": {"json"}}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", "/"+endpoint+"?"+q.Encode(), nil))
		return parseJSONResponse(w)
	}

	It("authenticates ping with only the key", func() {
		resp := doKeyReq("ping", key)

		Expect(resp.Status).To(Equal(responses.StatusOK))
	})

	It("reports the key owner in tokenInfo", func() {
		resp := doKeyReq("tokenInfo", key)

		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.TokenInfo).ToNot(BeNil())
		Expect(resp.TokenInfo.Username).To(Equal(regularUser.UserName))
	})

	It("rejects an unknown key with error 44", func() {
		resp := doKeyReq("ping", "nav_unknown")

		Expect(resp.Status).To(Equal(responses.StatusFailed))
		Expect(resp.Error).ToNot(BeNil())
		Expect(resp.Error.Code).To(Equal(int32(44)))
	})
})
