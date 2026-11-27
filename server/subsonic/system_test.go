package subsonic

import (
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TokenInfo", func() {
	var ctx context.Context
	BeforeEach(func() {
		ctx = GinkgoT().Context()
	})

	It("returns the authenticated username", func() {
		api := &Router{}
		r := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tokenInfo", nil)
		r = r.WithContext(request.WithUser(r.Context(), model.User{UserName: "deluan"}))

		resp, err := api.TokenInfo(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.TokenInfo.Username).To(Equal("deluan"))
	})
})
