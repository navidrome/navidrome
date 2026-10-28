package subsonic

import (
	"net/http/httptest"

	"github.com/SunilSherkhane/navidrome/model"
	"github.com/SunilSherkhane/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TokenInfo", func() {
	It("returns the authenticated username", func() {
		api := &Router{}
		r := httptest.NewRequest("GET", "/tokenInfo", nil)
		r = r.WithContext(request.WithUser(r.Context(), model.User{UserName: "deluan"}))

		resp, err := api.TokenInfo(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.TokenInfo.Username).To(Equal("deluan"))
	})
})
