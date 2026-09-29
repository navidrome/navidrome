package apiv1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/api"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GET /server", func() {
	var ctx context.Context
	var ds *tests.MockDataStore
	var users *tests.MockedUserRepo

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		users = tests.CreateMockUserRepo()
		ds = &tests.MockDataStore{MockedUser: users}
	})

	get := func() (*httptest.ResponseRecorder, ServerInfo) {
		w := serve(New(ds), httptest.NewRequest(http.MethodGet, "/api/v1/server", nil))
		var info ServerInfo
		if w.Code == http.StatusOK {
			ExpectWithOffset(1, json.Unmarshal(w.Body.Bytes(), &info)).To(Succeed())
		}
		return w, info
	}

	It("describes the server with setupRequired when there are no users", func() {
		w, info := get()
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Header().Get("Content-Type")).To(HavePrefix("application/json"))
		Expect(info.Name).To(Equal("Navidrome"))
		Expect(info.ServerVersion).To(Equal(consts.Version))
		Expect(info.SpecVersion).To(Equal(api.SpecVersion()))
		Expect(info.SetupRequired).To(BeTrue())
		Expect(info.LoginMethods.Password).ToNot(BeNil())
		Expect(w.Body.String()).To(ContainSubstring(`"loginMethods":{"password":{}}`))
	})

	It("reports setupRequired=false once a user exists", func() {
		Expect(users.Put(ctx, &model.User{ID: "u1", UserName: "admin", IsAdmin: true})).To(Succeed())
		_, info := get()
		Expect(info.SetupRequired).To(BeFalse())
	})

	It("returns a 500 problem when the user count fails", func() {
		users.Error = errors.New("db down")
		w, _ := get()
		Expect(w.Code).To(Equal(http.StatusInternalServerError))
		Expect(decodeProblem(w).Code).To(Equal(ProblemCodeInternal))
	})
})

var _ = Describe("GET /capabilities", func() {
	var ctx context.Context
	var api testClient

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		DeferCleanup(configtest.SetupConfig())
		resetDB()
		api = testClient{ctx: ctx, router: New(realDS)}
	})

	It("needs a grant", func() {
		w := api.call(http.MethodGet, "/api/v1/capabilities", "", nil)
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
	})

	It("lists core and password for any valid grant, even one with no scopes", func() {
		api.setup()
		gc := api.login([]string{})
		Expect(gc.Grant.Scopes).To(BeEmpty())

		w := api.call(http.MethodGet, "/api/v1/capabilities", gc.Secret, nil)
		Expect(w.Code).To(Equal(http.StatusOK))
		var caps Capabilities
		decodeJSON(w, &caps)
		Expect(caps.Core.Version).To(Equal(1))
		Expect(caps.Password.Version).To(Equal(1))
	})
})
