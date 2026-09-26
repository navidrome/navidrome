package apiv1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/conf/configtest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GET /capabilities", func() {
	var ctx context.Context
	var router *Router

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		DeferCleanup(configtest.SetupConfig())
		resetDB()
		router = New(realDS)
	})

	It("needs a token", func() {
		w := serve(router, httptest.NewRequestWithContext(ctx, http.MethodGet, "/api/v1/capabilities", nil))
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
	})

	It("lists core and password for any valid token, even one with no scopes", func() {
		body, _ := json.Marshal(map[string]any{"username": "admin", "password": "pw", "client": "c"})
		setupReq := httptest.NewRequestWithContext(ctx, http.MethodPost, "/api/v1/auth/setup", bytes.NewReader(body))
		setupReq.Header.Set("Content-Type", "application/json")
		var gc GrantCreated
		Expect(json.Unmarshal(serve(router, setupReq).Body.Bytes(), &gc)).To(Succeed())

		tokReq := httptest.NewRequestWithContext(ctx, http.MethodPost, "/api/v1/auth/token", bytes.NewReader([]byte(`{"scopes":[]}`)))
		tokReq.Header.Set("Content-Type", "application/json")
		tokReq.Header.Set("Authorization", "Bearer "+gc.Secret)
		var at AccessToken
		Expect(json.Unmarshal(serve(router, tokReq).Body.Bytes(), &at)).To(Succeed())

		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/api/v1/capabilities", nil)
		req.Header.Set("Authorization", "Bearer "+at.AccessToken)
		w := serve(router, req)
		Expect(w.Code).To(Equal(http.StatusOK))
		var caps Capabilities
		Expect(json.Unmarshal(w.Body.Bytes(), &caps)).To(Succeed())
		Expect(caps.Core.Version).To(Equal(1))
		Expect(caps.Password.Version).To(Equal(1))
	})
})
