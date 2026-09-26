package apiv1

import (
	"context"
	"net/http"

	"github.com/navidrome/navidrome/conf/configtest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GET /capabilities", func() {
	var ctx context.Context
	var api testClient

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		DeferCleanup(configtest.SetupConfig())
		resetDB()
		api = testClient{ctx: ctx, router: New(realDS)}
	})

	It("needs a token", func() {
		w := api.call(http.MethodGet, "/api/v1/capabilities", "", nil)
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
	})

	It("lists core and password for any valid token, even one with no scopes", func() {
		gc := api.setup()
		at := api.mint(gc.Secret, map[string]any{"scopes": []string{}})

		w := api.call(http.MethodGet, "/api/v1/capabilities", at.AccessToken, nil)
		Expect(w.Code).To(Equal(http.StatusOK))
		var caps Capabilities
		decodeJSON(w, &caps)
		Expect(caps.Core.Version).To(Equal(1))
		Expect(caps.Password.Version).To(Equal(1))
	})
})
