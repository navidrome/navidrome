package apiv1

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/auth"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("auth endpoints", func() {
	var ctx context.Context
	var api testClient

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		DeferCleanup(configtest.SetupConfig())
		conf.Server.AuthRequestLimit = 0
		resetDB()
		api = testClient{ctx: ctx, router: New(realDS)}
	})

	It("lets exactly one of a v1 setup and a v0 first-admin creation win", func() {
		var wg sync.WaitGroup
		var v1Code int
		var v0Err error
		wg.Add(2)
		go func() {
			defer GinkgoRecover()
			defer wg.Done()
			v1Code = api.call(http.MethodPost, "/api/v1/auth/setup", "", creds("v1admin", "pw")).Code
		}()
		go func() {
			defer GinkgoRecover()
			defer wg.Done()
			_, v0Err = auth.CreateFirstAdmin(ctx, realDS, "v0admin", "pw", nil) // what v0 /auth/createAdmin runs
		}()
		wg.Wait()
		Expect(realDS.User().CountAll(ctx)).To(Equal(int64(1)))
		Expect(v1Code).To(Or(Equal(http.StatusCreated), Equal(http.StatusConflict)))
		Expect(v1Code == http.StatusCreated).ToNot(Equal(v0Err == nil), "exactly one must win")
	})

	It("sets up the first admin once, then answers 409 setup_complete", func() {
		gc := api.setup()
		Expect(gc.Secret).To(HavePrefix("ndg_"))
		Expect(gc.User.IsAdmin).To(BeTrue())
		Expect(gc.Grant.Provider).To(Equal("setup"))
		Expect(gc.Grant.Current).To(BeTrue())

		w := api.call(http.MethodPost, "/api/v1/auth/setup", "", creds("second", "pw"))
		Expect(w.Code).To(Equal(http.StatusConflict))
		Expect(decodeProblem(w).Code).To(Equal(ProblemCodeSetupComplete))
	})

	It("logs in and uses the grant secret on a scoped endpoint", func() {
		api.setup()
		w := api.call(http.MethodPost, "/api/v1/auth/login", "", creds("ADMIN", "pw"))
		Expect(w.Code).To(Equal(http.StatusOK), w.Body.String())
		var gc GrantCreated
		decodeJSON(w, &gc)
		Expect(gc.User.PasswordChangeable).To(BeTrue())

		w = api.call(http.MethodGet, "/api/v1/auth/grants", gc.Secret, nil)
		Expect(w.Code).To(Equal(http.StatusOK), w.Body.String())
		var list GrantList
		decodeJSON(w, &list)
		Expect(list.Total).To(Equal(2))
		Expect(list.Limit).To(Equal(100))
	})

	It("fails login the same way for an unknown user and a wrong password, with a Bearer challenge", func() {
		api.setup()
		a := api.call(http.MethodPost, "/api/v1/auth/login", "", creds("admin", "wrong"))
		b := api.call(http.MethodPost, "/api/v1/auth/login", "", creds("ghost", "pw"))
		Expect(a.Code).To(Equal(http.StatusUnauthorized))
		Expect(a.Header().Get("WWW-Authenticate")).To(Equal("Bearer"))
		Expect(a.Body.String()).To(Equal(b.Body.String()))
	})

	It("treats missing scopes as all scopes, and [] as no scopes", func() {
		api.setup()
		all := api.login(nil)
		Expect(all.Grant.Scopes).To(ConsistOf(ScopeAll))
		Expect(api.call(http.MethodGet, "/api/v1/auth/grants", all.Secret, nil).Code).To(Equal(http.StatusOK))

		none := api.login([]string{})
		Expect(none.Grant.Scopes).To(BeEmpty())
		w := api.call(http.MethodGet, "/api/v1/auth/grants", none.Secret, nil)
		Expect(w.Code).To(Equal(http.StatusForbidden))
		Expect(w.Header().Get("WWW-Authenticate")).To(Equal(`Bearer error="insufficient_scope", scope="read"`))
		Expect(decodeProblem(w).Code).To(Equal(ProblemCodeInsufficientScope))
	})

	It("drops unknown requested scopes instead of rejecting them", func() {
		api.setup()
		gc := api.login([]string{"read", "playlists:write"})
		Expect(gc.Grant.Scopes).To(ConsistOf(ScopeRead))
	})

	It("does not let a grant without read log out or revoke grants", func() {
		gc := api.setup()
		narrow := api.login([]string{"password"})
		Expect(api.call(http.MethodPost, "/api/v1/auth/logout", narrow.Secret, nil).Code).To(Equal(http.StatusForbidden))
		Expect(api.call(http.MethodDelete, "/api/v1/auth/grants/"+gc.Grant.Id, narrow.Secret, nil).Code).To(Equal(http.StatusForbidden))
	})

	It("logs out: the secret stops at once and logoutUrl is null", func() {
		gc := api.setup()
		w := api.call(http.MethodPost, "/api/v1/auth/logout", gc.Secret, nil)
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Body.String()).To(ContainSubstring(`"logoutUrl":null`))

		w = api.call(http.MethodGet, "/api/v1/auth/grants", gc.Secret, nil)
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
		Expect(w.Header().Get("WWW-Authenticate")).To(Equal(`Bearer error="invalid_token"`))
	})

	It("revokes another grant of the caller, whose secret then stops at once", func() {
		gc := api.setup()
		other := api.login(nil)
		Expect(api.call(http.MethodDelete, "/api/v1/auth/grants/"+other.Grant.Id, gc.Secret, nil).Code).To(Equal(http.StatusNoContent))
		Expect(api.call(http.MethodGet, "/api/v1/auth/grants", other.Secret, nil).Code).To(Equal(http.StatusUnauthorized))
		Expect(api.call(http.MethodGet, "/api/v1/auth/grants", gc.Secret, nil).Code).To(Equal(http.StatusOK))
	})

	It("challenges with invalid_token when the grant is revoked while a password change runs", func() {
		gc := api.setup()
		Expect(realDS.Grant().DeleteForUser(ctx, gc.User.Id, gc.Grant.Id)).To(Succeed())

		w := api.call(http.MethodPost, "/api/v1/auth/password", gc.Secret, map[string]any{"currentPassword": "pw", "newPassword": "pw2"})
		Expect(w.Code).To(Equal(http.StatusUnauthorized), w.Body.String())
		Expect(w.Header().Get("WWW-Authenticate")).To(Equal(`Bearer error="invalid_token"`))
	})

	It("rejects a case-variant scopes key that would widen an explicit empty subset", func() {
		api.setup()
		w := api.callRaw(http.MethodPost, "/api/v1/auth/login", "", `{"username":"admin","password":"pw","client":"c","scopes":[],"Scopes":null}`)
		Expect(w.Code).To(Equal(http.StatusBadRequest), w.Body.String())
		p := decodeProblem(w)
		Expect(p.Code).To(Equal(ProblemCodeValidation))
		Expect(*p.Errors).To(ConsistOf(ValidationError{Field: "Scopes", Message: "must match the field name exactly"}))
	})

	It("rejects a case-variant client key that would skip its length limit", func() {
		api.setup()
		body := `{"username":"admin","password":"pw","client":"ok","Client":"` + strings.Repeat("x", 60_000) + `"}`
		w := api.callRaw(http.MethodPost, "/api/v1/auth/login", "", body)
		Expect(w.Code).To(Equal(http.StatusBadRequest), w.Body.String())
		Expect(*decodeProblem(w).Errors).To(ConsistOf(ValidationError{Field: "Client", Message: "must match the field name exactly"}))
	})

	DescribeTable("rejects a body with data after its JSON value, without echoing it",
		func(body string) {
			api.setup()
			w := api.callRaw(http.MethodPost, "/api/v1/auth/login", "", body)
			Expect(w.Code).To(Equal(http.StatusBadRequest), w.Body.String())
			p := decodeProblem(w)
			Expect(p.Code).To(Equal(ProblemCodeValidation))
			Expect(*p.Errors).To(ConsistOf(ValidationError{Field: "", Message: "must be a single JSON value"}))
			Expect(w.Body.String()).ToNot(ContainSubstring("hunter2"))
		},
		Entry("a trailing byte", `{"username":"a","password":"hunter2","client":"c"}x`),
		Entry("a second value", `{"username":"a","password":"hunter2","client":"c"} {}`),
	)

	It("checks the login body even when Content-Type has a repeated parameter", func() {
		api.setup()
		body := `{"username":"admin","password":"pw","client":"c","scopes":[],"Scopes":null}`
		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json; a=1; a=2")
		w := serve(api.router, req)
		Expect(w.Code).To(Equal(http.StatusBadRequest), w.Body.String())
		Expect(*decodeProblem(w).Errors).To(ConsistOf(ValidationError{Field: "Scopes", Message: "must match the field name exactly"}))
	})

	It("marks responses carrying a grant secret no-store", func() {
		w := api.call(http.MethodPost, "/api/v1/auth/setup", "", creds("admin", "pw"))
		Expect(w.Code).To(Equal(http.StatusCreated))
		Expect(w.Header().Get("Cache-Control")).To(Equal("no-store"))
		var gc GrantCreated
		decodeJSON(w, &gc)

		w = api.call(http.MethodPost, "/api/v1/auth/login", "", creds("admin", "pw"))
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Header().Get("Cache-Control")).To(Equal("no-store"))

		w = api.call(http.MethodGet, "/api/v1/auth/grants", gc.Secret, nil)
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Header().Get("Cache-Control")).To(BeEmpty())
	})

	It("answers 404 for a grant id the caller does not own, and 400 for an over-long id", func() {
		gc := api.setup()
		Expect(api.call(http.MethodDelete, "/api/v1/auth/grants/does-not-exist", gc.Secret, nil).Code).To(Equal(http.StatusNotFound))
		w := api.call(http.MethodDelete, "/api/v1/auth/grants/"+strings.Repeat("x", 65), gc.Secret, nil)
		Expect(w.Code).To(Equal(http.StatusBadRequest))
		Expect(*decodeProblem(w).Errors).To(ConsistOf(ValidationError{Field: "id", Message: "is too long"}))
	})

	It("changes the password, keeping the caller and revoking the rest", func() {
		gc := api.setup()
		other := api.login(nil)

		w := api.call(http.MethodPost, "/api/v1/auth/password", gc.Secret, map[string]any{"currentPassword": "pw", "newPassword": "pw2"})
		Expect(w.Code).To(Equal(http.StatusNoContent), w.Body.String())

		Expect(api.call(http.MethodGet, "/api/v1/auth/grants", gc.Secret, nil).Code).To(Equal(http.StatusOK))
		Expect(api.call(http.MethodGet, "/api/v1/auth/grants", other.Secret, nil).Code).To(Equal(http.StatusUnauthorized))
	})

	It("keeps every grant when revokeOtherGrants is false", func() {
		gc := api.setup()
		other := api.login(nil)

		body := map[string]any{"currentPassword": "pw", "newPassword": "pw2", "revokeOtherGrants": false}
		w := api.call(http.MethodPost, "/api/v1/auth/password", gc.Secret, body)
		Expect(w.Code).To(Equal(http.StatusNoContent), w.Body.String())

		Expect(api.call(http.MethodGet, "/api/v1/auth/grants", gc.Secret, nil).Code).To(Equal(http.StatusOK))
		Expect(api.call(http.MethodGet, "/api/v1/auth/grants", other.Secret, nil).Code).To(Equal(http.StatusOK))
	})

	It("reports a wrong current password as a field error", func() {
		gc := api.setup()
		w := api.call(http.MethodPost, "/api/v1/auth/password", gc.Secret, map[string]any{"currentPassword": "nope", "newPassword": "pw2"})
		Expect(w.Code).To(Equal(http.StatusBadRequest))
		p := decodeProblem(w)
		Expect(*p.Errors).To(ConsistOf(ValidationError{Field: "currentPassword", Message: "is incorrect"}))
	})

	DescribeTable("rejects bad credential bodies with a field error and no echo",
		func(body map[string]any, field string) {
			w := api.call(http.MethodPost, "/api/v1/auth/setup", "", body)
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
			w := serve(api.router, req)
			Expect(w.Code).To(Equal(http.StatusRequestEntityTooLarge))
			Expect(decodeProblem(w).Code).To(Equal(ProblemCodePayloadTooLarge))
		},
		Entry("with a declared length", func(s string) io.Reader { return strings.NewReader(s) }),
		// io.MultiReader hides the length, so the request has ContentLength -1, like a chunked upload.
		Entry("with no declared length", func(s string) io.Reader { return io.MultiReader(strings.NewReader(s)) }),
	)
})
