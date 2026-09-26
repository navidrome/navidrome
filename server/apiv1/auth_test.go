package apiv1

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("auth endpoints", func() {
	var ctx context.Context
	var router *Router

	call := func(method, path, bearer string, body any) *httptest.ResponseRecorder {
		var req *http.Request
		if body != nil {
			b, _ := json.Marshal(body)
			req = httptest.NewRequestWithContext(ctx, method, path, bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")
		} else {
			req = httptest.NewRequestWithContext(ctx, method, path, nil)
		}
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		return serve(router, req)
	}

	creds := func(user, pw string) map[string]any {
		return map[string]any{"username": user, "password": pw, "client": "TestApp", "clientVersion": "1.0"}
	}

	decode := func(w *httptest.ResponseRecorder, v any) {
		ExpectWithOffset(1, json.Unmarshal(w.Body.Bytes(), v)).To(Succeed(), w.Body.String())
	}

	setup := func() GrantCreated {
		w := call(http.MethodPost, "/api/v1/auth/setup", "", creds("admin", "pw"))
		ExpectWithOffset(1, w.Code).To(Equal(http.StatusCreated), w.Body.String())
		var gc GrantCreated
		decode(w, &gc)
		return gc
	}

	mint := func(secret string, body any) AccessToken {
		w := call(http.MethodPost, "/api/v1/auth/token", secret, body)
		ExpectWithOffset(1, w.Code).To(Equal(http.StatusOK), w.Body.String())
		var at AccessToken
		decode(w, &at)
		return at
	}

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		DeferCleanup(configtest.SetupConfig())
		conf.Server.AuthRequestLimit = 0
		resetDB()
		router = New(realDS)
	})

	It("lets exactly one of a v1 setup and a v0 first-admin creation win", func() {
		var wg sync.WaitGroup
		var v1Code int
		var v0Err error
		wg.Add(2)
		go func() {
			defer GinkgoRecover()
			defer wg.Done()
			v1Code = call(http.MethodPost, "/api/v1/auth/setup", "", creds("v1admin", "pw")).Code
		}()
		go func() {
			defer GinkgoRecover()
			defer wg.Done()
			v0Err = realDS.WithTxImmediate(func(tx model.DataStore) error { // what v0 /auth/createAdmin runs
				_, err := auth.CreateFirstAdmin(ctx, tx, "v0admin", "pw")
				return err
			})
		}()
		wg.Wait()
		Expect(realDS.User().CountAll(ctx)).To(Equal(int64(1)))
		Expect(v1Code == http.StatusCreated).ToNot(Equal(v0Err == nil), "exactly one must win")
	})

	It("sets up the first admin once, then answers 409 setup_complete", func() {
		gc := setup()
		Expect(gc.Secret).To(HavePrefix("ndg_"))
		Expect(gc.User.IsAdmin).To(BeTrue())
		Expect(gc.Grant.Provider).To(Equal("setup"))
		Expect(gc.Grant.Current).To(BeTrue())

		w := call(http.MethodPost, "/api/v1/auth/setup", "", creds("second", "pw"))
		Expect(w.Code).To(Equal(http.StatusConflict))
		Expect(decodeProblem(w).Code).To(Equal(ProblemCodeSetupComplete))
	})

	It("logs in, mints a token, and uses it on a scoped endpoint", func() {
		setup()
		w := call(http.MethodPost, "/api/v1/auth/login", "", creds("ADMIN", "pw"))
		Expect(w.Code).To(Equal(http.StatusOK), w.Body.String())
		var gc GrantCreated
		decode(w, &gc)
		Expect(gc.User.PasswordChangeable).To(BeTrue())

		at := mint(gc.Secret, nil)
		Expect(at.TokenType).To(Equal(AccessTokenTokenTypeBearer))
		Expect(at.ExpiresIn).To(Equal(3600))

		w = call(http.MethodGet, "/api/v1/auth/grants", at.AccessToken, nil)
		Expect(w.Code).To(Equal(http.StatusOK), w.Body.String())
		var list GrantList
		decode(w, &list)
		Expect(list.Total).To(Equal(2))
		Expect(list.Limit).To(Equal(100))
	})

	It("fails login the same way for an unknown user and a wrong password, with a Bearer challenge", func() {
		setup()
		a := call(http.MethodPost, "/api/v1/auth/login", "", creds("admin", "wrong"))
		b := call(http.MethodPost, "/api/v1/auth/login", "", creds("ghost", "pw"))
		Expect(a.Code).To(Equal(http.StatusUnauthorized))
		Expect(a.Header().Get("WWW-Authenticate")).To(Equal("Bearer"))
		Expect(a.Body.String()).To(Equal(b.Body.String()))
	})

	It("treats no body and {} as all scopes, and [] as no scopes", func() {
		gc := setup()
		all := mint(gc.Secret, nil)
		Expect(all.Scopes).To(ConsistOf(ScopeRead, ScopePassword))
		Expect(mint(gc.Secret, map[string]any{}).Scopes).To(ConsistOf(ScopeRead, ScopePassword))

		none := mint(gc.Secret, map[string]any{"scopes": []string{}})
		Expect(none.Scopes).To(BeEmpty())
		w := call(http.MethodGet, "/api/v1/auth/grants", none.AccessToken, nil)
		Expect(w.Code).To(Equal(http.StatusForbidden))
		Expect(decodeProblem(w).Code).To(Equal(ProblemCodeInsufficientScope))
	})

	It("drops unknown requested scopes instead of rejecting them", func() {
		gc := setup()
		at := mint(gc.Secret, map[string]any{"scopes": []string{"read", "playlists:write"}})
		Expect(at.Scopes).To(ConsistOf(ScopeRead))
	})

	It("does not let a token without read log out or revoke grants", func() {
		gc := setup()
		narrow := mint(gc.Secret, map[string]any{"scopes": []string{"password"}})
		Expect(call(http.MethodPost, "/api/v1/auth/logout", narrow.AccessToken, nil).Code).To(Equal(http.StatusForbidden))
		Expect(call(http.MethodDelete, "/api/v1/auth/grants/"+gc.Grant.Id, narrow.AccessToken, nil).Code).To(Equal(http.StatusForbidden))
	})

	It("logs out: the token stops at once and logoutUrl is null", func() {
		gc := setup()
		at := mint(gc.Secret, nil)
		w := call(http.MethodPost, "/api/v1/auth/logout", at.AccessToken, nil)
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Body.String()).To(ContainSubstring(`"logoutUrl":null`))

		w = call(http.MethodGet, "/api/v1/auth/grants", at.AccessToken, nil)
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
		Expect(call(http.MethodPost, "/api/v1/auth/token", gc.Secret, nil).Code).To(Equal(http.StatusUnauthorized))
	})

	It("answers 404 for a grant id the caller does not own, and 400 for an over-long id", func() {
		gc := setup()
		tok := mint(gc.Secret, nil).AccessToken
		Expect(call(http.MethodDelete, "/api/v1/auth/grants/does-not-exist", tok, nil).Code).To(Equal(http.StatusNotFound))
		w := call(http.MethodDelete, "/api/v1/auth/grants/"+strings.Repeat("x", 65), tok, nil)
		Expect(w.Code).To(Equal(http.StatusBadRequest))
		Expect(*decodeProblem(w).Errors).To(ConsistOf(ValidationError{Field: "id", Message: "is too long"}))
	})

	It("changes the password, keeping the caller and revoking the rest", func() {
		gc := setup()
		otherLogin := call(http.MethodPost, "/api/v1/auth/login", "", creds("admin", "pw"))
		var other GrantCreated
		decode(otherLogin, &other)
		at := mint(gc.Secret, nil)

		w := call(http.MethodPost, "/api/v1/auth/password", at.AccessToken, map[string]any{"currentPassword": "pw", "newPassword": "pw2"})
		Expect(w.Code).To(Equal(http.StatusNoContent), w.Body.String())

		Expect(call(http.MethodGet, "/api/v1/auth/grants", at.AccessToken, nil).Code).To(Equal(http.StatusOK))
		Expect(call(http.MethodPost, "/api/v1/auth/token", other.Secret, nil).Code).To(Equal(http.StatusUnauthorized))
	})

	It("reports a wrong current password as a field error", func() {
		gc := setup()
		at := mint(gc.Secret, nil)
		w := call(http.MethodPost, "/api/v1/auth/password", at.AccessToken, map[string]any{"currentPassword": "nope", "newPassword": "pw2"})
		Expect(w.Code).To(Equal(http.StatusBadRequest))
		p := decodeProblem(w)
		Expect(*p.Errors).To(ConsistOf(ValidationError{Field: "currentPassword", Message: "is incorrect"}))
	})

	DescribeTable("rejects bad credential bodies with a field error and no echo",
		func(body map[string]any, field string) {
			w := call(http.MethodPost, "/api/v1/auth/setup", "", body)
			Expect(w.Code).To(Equal(http.StatusBadRequest), w.Body.String())
			p := decodeProblem(w)
			Expect(p.Code).To(Equal(ProblemCodeValidation))
			Expect(*p.Errors).To(ContainElement(HaveField("Field", field)))
			Expect(w.Body.String()).ToNot(ContainSubstring("hunter2"))
		},
		Entry("missing client", map[string]any{"username": "a", "password": "hunter2"}, "client"),
		Entry("empty password", map[string]any{"username": "a", "password": "", "client": "hunter2"}, "password"),
		Entry("client too long", map[string]any{"username": "a", "password": "hunter2", "client": strings.Repeat("x", 65)}, "client"),
		Entry("bad scope format", map[string]any{"username": "a", "password": "hunter2", "client": "c", "scopes": []string{"NOT OK"}}, "scopes.0"),
	)

	DescribeTable("rejects a body over 1 MiB with 413",
		func(body func(string) io.Reader) {
			big := `{"username":"a","password":"` + strings.Repeat("a", maxBodyBytes) + `","client":"c"}`
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/api/v1/auth/login", body(big))
			req.Header.Set("Content-Type", "application/json")
			w := serve(router, req)
			Expect(w.Code).To(Equal(http.StatusRequestEntityTooLarge))
			Expect(decodeProblem(w).Code).To(Equal(ProblemCodePayloadTooLarge))
		},
		Entry("with a declared length", func(s string) io.Reader { return strings.NewReader(s) }),
		// io.MultiReader hides the length, so the request has ContentLength -1, like a chunked upload.
		Entry("with no declared length", func(s string) io.Reader { return io.MultiReader(strings.NewReader(s)) }),
	)
})
