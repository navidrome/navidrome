package e2e

import (
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("QuickConnect", func() {
	BeforeEach(func() {
		setupTestDB()
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Jellyfin.QuickConnect = true
	})

	clientReq := func(deviceID, method, path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Authorization", `MediaBrowser Client="Finamp", Device="Pixel 7", DeviceId="`+deviceID+`", Version="1.0"`)
		r.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, r)
		return w
	}
	initiate := func() dto.QuickConnectResult {
		var pending dto.QuickConnectResult
		parseInto(clientReq("new-device", "POST", "/QuickConnect/Initiate", ""), &pending)
		Expect(pending.Authenticated).To(BeFalse())
		return pending
	}
	redeem := func(deviceID, secret string) *httptest.ResponseRecorder {
		return clientReq(deviceID, "POST", "/Users/AuthenticateWithQuickConnect", `{"Secret":"`+secret+`"}`)
	}

	It("signs a new client in with a code approved by a signed-in user", func() {
		Expect(rawReq("GET", "/QuickConnect/Enabled", "").Body.String()).To(MatchJSON("true"))
		pending := initiate()

		Expect(postAs(regularUser, "/QuickConnect/Authorize?Code="+pending.Code, "").Code).To(Equal(http.StatusOK))

		var status dto.QuickConnectResult
		parseInto(rawReq("GET", "/QuickConnect/Connect?Secret="+pending.Secret, ""), &status)
		Expect(status.Authenticated).To(BeTrue())

		// Android TV redeems with a different DeviceId than it initiated with.
		w := redeem("other-device", pending.Secret)
		Expect(w.Code).To(Equal(http.StatusOK))
		var res dto.AuthenticationResult
		parseInto(w, &res)
		Expect(res.User.Name).To(Equal(regularUser.UserName))
		Expect(res.SessionInfo).ToNot(BeNil())
		Expect(res.SessionInfo.DeviceId).To(Equal("other-device"))

		r := httptest.NewRequest("GET", "/Users/Me", nil)
		r.Header.Set("X-Emby-Token", res.AccessToken)
		me := httptest.NewRecorder()
		router.ServeHTTP(me, r)
		Expect(me.Code).To(Equal(http.StatusOK))

		Expect(redeem("new-device", pending.Secret).Code).To(Equal(http.StatusNotFound))
	})

	It("lets an admin approve a code for another user", func() {
		pending := initiate()
		Expect(post("/QuickConnect/Authorize?Code="+pending.Code+"&UserId="+enc(regularUser.ID), "").Code).
			To(Equal(http.StatusOK))

		var res dto.AuthenticationResult
		parseInto(redeem("new-device", pending.Secret), &res)
		Expect(res.User.Name).To(Equal(regularUser.UserName))
	})

	It("requires authentication to approve a code", func() {
		pending := initiate()
		Expect(rawReq("POST", "/QuickConnect/Authorize?Code="+pending.Code, "").Code).To(Equal(http.StatusUnauthorized))
	})

	It("answers 401 on every Quick Connect call when disabled", func() {
		conf.Server.Jellyfin.QuickConnect = false
		Expect(rawReq("GET", "/QuickConnect/Enabled", "").Body.String()).To(MatchJSON("false"))
		Expect(clientReq("new-device", "POST", "/QuickConnect/Initiate", "").Code).To(Equal(http.StatusUnauthorized))
		Expect(rawReq("GET", "/QuickConnect/Connect?Secret=x", "").Code).To(Equal(http.StatusUnauthorized))
		Expect(post("/QuickConnect/Authorize?Code=123456", "").Code).To(Equal(http.StatusUnauthorized))
		Expect(redeem("new-device", "x").Code).To(Equal(http.StatusUnauthorized))
	})
})
