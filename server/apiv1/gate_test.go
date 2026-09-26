package apiv1

import (
	"bytes"
	"context"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/apiauth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const gateSpec = `
openapi: 3.0.3
info: {title: t, version: "1"}
paths:
  /open:
    get: {operationId: open, x-module: core, security: [], responses: {'200': {description: ok}}}
  /things/{id}:
    get:
      operationId: getThing
      x-module: core
      x-scope: read
      security: [{bearerAuth: []}]
      parameters: [{name: id, in: path, required: true, schema: {type: string, maxLength: 3}}]
      responses: {'200': {description: ok}}
  /things:
    post:
      operationId: createThing
      x-module: password
      x-scope: password
      security: [{bearerAuth: []}]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [name]
              properties: {name: {type: string, maxLength: 5}}
      responses: {'200': {description: ok}}
  /caps:
    get: {operationId: caps, x-module: core, security: [{bearerAuth: []}], responses: {'200': {description: ok}}}
  /mint:
    post: {operationId: mint, x-module: core, security: [{grantAuth: []}], responses: {'200': {description: ok}}}
  /limited:
    post: {operationId: limited, x-module: core, security: [], responses: {'200': {description: ok}}}
components:
  securitySchemes:
    bearerAuth: {type: http, scheme: bearer}
    grantAuth: {type: http, scheme: bearer}
`

type fakeAuth struct {
	principal *apiauth.Principal
	err       error
	gotToken  string
	gotSecret string
	gotIP     string
}

func (f *fakeAuth) Authenticate(_ context.Context, token, ip string) (*apiauth.Principal, error) {
	f.gotToken, f.gotIP = token, ip
	return f.principal, f.err
}

func (f *fakeAuth) ResolveGrant(_ context.Context, secret, ip string) (*apiauth.Principal, error) {
	f.gotSecret, f.gotIP = secret, ip
	return f.principal, f.err
}

var testGateRules = gateRules{
	limited:  map[string]bool{"limited": true},
	noScope:  map[string]bool{"caps": true},
	grantOps: map[string]bool{"mint": true},
	noStore:  map[string]bool{"mint": true},
}

var _ = Describe("spec gate", func() {
	var ctx context.Context
	var fa *fakeAuth
	var mux *chi.Mux
	var g *gate
	var reached string

	build := func(spec string) (*chi.Mux, error) {
		doc, err := openapi3.NewLoader().LoadFromData([]byte(spec))
		Expect(err).ToNot(HaveOccurred())
		m := chi.NewRouter()
		g, err = newGate(doc, m, fa, testGateRules)
		if err != nil {
			return nil, err
		}
		m.Use(headAsGet(m), g.handler)
		ok := func(name string) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				reached = name
				if p, found := apiauth.PrincipalFrom(r.Context()); found {
					w.Header().Set("X-User", p.User.ID)
				}
				w.WriteHeader(http.StatusOK)
			}
		}
		m.Get("/open", ok("open"))
		m.Get("/things/{id}", ok("getThing"))
		m.Post("/things", ok("createThing"))
		m.Get("/caps", ok("caps"))
		m.Post("/mint", ok("mint"))
		m.Post("/limited", ok("limited"))
		return m, nil
	}

	do := func(method, path, auth, body string) *httptest.ResponseRecorder {
		var req *http.Request
		if body != "" {
			req = httptest.NewRequestWithContext(ctx, method, path, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
		} else {
			req = httptest.NewRequestWithContext(ctx, method, path, nil)
		}
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		return w
	}

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		reached = ""
		fa = &fakeAuth{principal: &apiauth.Principal{User: model.User{ID: "u1"}, GrantID: "g1", Scopes: []string{"read"}}}
		var err error
		mux, err = build(gateSpec)
		Expect(err).ToNot(HaveOccurred())
	})

	It("lets public operations through without a token", func() {
		Expect(do(http.MethodGet, "/open", "", "").Code).To(Equal(http.StatusOK))
		Expect(reached).To(Equal("open"))
	})

	It("requires a token, with a Bearer challenge", func() {
		w := do(http.MethodGet, "/things/1", "", "")
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
		Expect(w.Header().Get("WWW-Authenticate")).To(Equal("Bearer"))
		Expect(decodeProblem(w).Code).To(Equal(ProblemCodeUnauthorized))
		Expect(reached).To(BeEmpty())
	})

	It("accepts the Bearer scheme in any case and trims spaces", func() {
		w := do(http.MethodGet, "/things/1", "bearer   tok-1 ", "")
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(fa.gotToken).To(Equal("tok-1"))
		Expect(w.Header().Get("X-User")).To(Equal("u1"))
	})

	It("maps an expired token to token_expired", func() {
		fa.err = apiauth.ErrTokenExpired
		w := do(http.MethodGet, "/things/1", "Bearer x", "")
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
		Expect(w.Header().Get("WWW-Authenticate")).To(Equal(`Bearer error="invalid_token"`))
		Expect(decodeProblem(w).Code).To(Equal(ProblemCodeTokenExpired))
	})

	It("maps other auth failures to unauthorized with invalid_token", func() {
		fa.err = model.ErrInvalidAuth
		w := do(http.MethodGet, "/things/1", "Bearer x", "")
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
		Expect(w.Header().Get("WWW-Authenticate")).To(Equal(`Bearer error="invalid_token"`))
		Expect(decodeProblem(w).Code).To(Equal(ProblemCodeUnauthorized))
	})

	It("rejects a token without the operation's scope", func() {
		w := do(http.MethodPost, "/things", "Bearer x", `{"name":"a"}`)
		Expect(w.Code).To(Equal(http.StatusForbidden))
		Expect(w.Header().Get("WWW-Authenticate")).To(Equal(`Bearer error="insufficient_scope", scope="password"`))
		Expect(decodeProblem(w).Code).To(Equal(ProblemCodeInsufficientScope))
	})

	It("lets any valid token through an operation with no x-scope", func() {
		fa.principal.Scopes = nil
		Expect(do(http.MethodGet, "/caps", "Bearer x", "").Code).To(Equal(http.StatusOK))
	})

	DescribeTable("passes the full client address to the authenticator, not the rate-limit /64",
		func(method, path string) {
			req := httptest.NewRequestWithContext(ctx, method, path, nil)
			req.RemoteAddr = "[2001:db8:1:2:3:4:5:6]:4321"
			req.Header.Set("Authorization", "Bearer x")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(fa.gotIP).To(Equal("2001:db8:1:2:3:4:5:6"))
		},
		Entry("access token", http.MethodGet, "/things/1"),
		Entry("grant", http.MethodPost, "/mint"),
	)

	It("marks only the listed operations' responses no-store, errors included", func() {
		Expect(do(http.MethodPost, "/mint", "Bearer ndg_secret", "").Header().Get("Cache-Control")).To(Equal("no-store"))
		fa.err = model.ErrInvalidAuth
		Expect(do(http.MethodPost, "/mint", "Bearer ndg_secret", "").Header().Get("Cache-Control")).To(Equal("no-store"))
		fa.err = nil
		Expect(do(http.MethodGet, "/things/1", "Bearer x", "").Header().Get("Cache-Control")).To(BeEmpty())
	})

	It("uses ResolveGrant for grantAuth operations", func() {
		Expect(do(http.MethodPost, "/mint", "Bearer ndg_secret", "").Code).To(Equal(http.StatusOK))
		Expect(fa.gotSecret).To(Equal("ndg_secret"))
		Expect(fa.gotToken).To(BeEmpty())
	})

	It("checks HEAD on a protected GET", func() {
		w := do(http.MethodHead, "/things/1", "", "")
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
	})

	It("turns an insufficient-scope error from Authenticate into a 403 challenge", func() {
		fa.err = apiauth.ErrInsufficientScope // e.g. a token carrying admin after demotion
		w := do(http.MethodGet, "/caps", "Bearer x", "")
		Expect(w.Code).To(Equal(http.StatusForbidden))
		Expect(w.Header().Get("WWW-Authenticate")).To(Equal(`Bearer error="insufficient_scope"`))
		Expect(decodeProblem(w).Code).To(Equal(ProblemCodeInsufficientScope))
	})

	It("names the operation's scope when Authenticate reports an insufficient scope", func() {
		fa.err = apiauth.ErrInsufficientScope
		w := do(http.MethodGet, "/things/1", "Bearer x", "")
		Expect(w.Code).To(Equal(http.StatusForbidden))
		Expect(w.Header().Get("WWW-Authenticate")).To(Equal(`Bearer error="insufficient_scope", scope="read"`))
	})

	It("looks routes up on the raw path, as chi dispatches them", func() {
		w := do(http.MethodGet, "/things/a%2Fb", "", "")
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
		Expect(reached).To(BeEmpty())

		root := chi.NewRouter()
		root.Mount("/music/api/v1", mux)
		w = httptest.NewRecorder()
		root.ServeHTTP(w, httptest.NewRequestWithContext(ctx, http.MethodGet, "/music/api/v1/things/a%2Fb", nil))
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
		Expect(reached).To(BeEmpty())
	})

	It("works when mounted under a base path", func() {
		root := chi.NewRouter()
		root.Mount("/music/api/v1", mux)
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/music/api/v1/things/1", nil)
		w := httptest.NewRecorder()
		root.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
	})

	It("authenticates before validating", func() {
		w := do(http.MethodPost, "/things", "", `{"name":"far-too-long"}`)
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
	})

	It("returns and logs sanitised validation errors that never echo the value", func() {
		logs := captureLogs()
		fa.principal.Scopes = []string{"password"}
		w := do(http.MethodPost, "/things", "Bearer x", `{"name":"hunter2-secret"}`)
		Expect(w.Code).To(Equal(http.StatusBadRequest))
		p := decodeProblem(w)
		Expect(p.Code).To(Equal(ProblemCodeValidation))
		Expect(*p.Errors).To(ConsistOf(ValidationError{Field: "name", Message: "is too long"}))
		Expect(w.Body.String()).ToNot(ContainSubstring("hunter2"))
		Expect(logs.String()).To(ContainSubstring("failed validation"))
		Expect(logs.String()).ToNot(ContainSubstring("hunter2"))
	})

	It("reports a missing required body field by name", func() {
		fa.principal.Scopes = []string{"password"}
		w := do(http.MethodPost, "/things", "Bearer x", `{}`)
		Expect(*decodeProblem(w).Errors).To(ConsistOf(ValidationError{Field: "name", Message: "is required"}))
	})

	It("validates path parameters", func() {
		w := do(http.MethodGet, "/things/toolong", "Bearer x", "")
		Expect(w.Code).To(Equal(http.StatusBadRequest))
		Expect(*decodeProblem(w).Errors).To(ConsistOf(ValidationError{Field: "id", Message: "is too long"}))
	})

	It("passes unknown paths through to the router's 404", func() {
		Expect(do(http.MethodGet, "/nope", "", "").Code).To(Equal(http.StatusNotFound))
	})

	It("fails closed for a routed pattern the spec does not know", func() {
		mux.Get("/extra", func(w http.ResponseWriter, r *http.Request) { reached = "extra" })
		w := do(http.MethodGet, "/extra", "", "")
		Expect(w.Code).To(Equal(http.StatusInternalServerError))
		Expect(reached).To(BeEmpty())
	})

	It("rate-limits the listed operations with a 429 problem", func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.AuthRequestLimit = 1
		var err error
		mux, err = build(gateSpec)
		Expect(err).ToNot(HaveOccurred())
		Expect(do(http.MethodPost, "/limited", "", "").Code).To(Equal(http.StatusOK))
		w := do(http.MethodPost, "/limited", "", "")
		Expect(w.Code).To(Equal(http.StatusTooManyRequests))
		Expect(w.Header().Get("Retry-After")).ToNot(BeEmpty())
		Expect(decodeProblem(w).Code).To(Equal(ProblemCodeRateLimited))
	})

	DescribeTable("refuses specs that break the security rules",
		func(bad string) {
			_, err := build(bad)
			Expect(err).To(HaveOccurred())
		},
		Entry("missing security", strings.Replace(gateSpec, "operationId: open, x-module: core, security: [],", "operationId: open, x-module: core,", 1)),
		Entry("scope not matching module", strings.Replace(gateSpec, "x-scope: read", "x-scope: password", 1)),
		Entry("unknown scope", strings.Replace(gateSpec, "x-scope: read", "x-scope: bogus", 1)),
		Entry("bearer without x-scope outside the allowlist", strings.Replace(gateSpec, "      x-scope: read\n", "", 1)),
		Entry("grantAuth outside the allowlist", strings.Replace(gateSpec, "operationId: limited, x-module: core, security: []", "operationId: limited, x-module: core, security: [{grantAuth: []}]", 1)),
		Entry("non-empty scope list on a bearer scheme", strings.Replace(gateSpec, "operationId: caps, x-module: core, security: [{bearerAuth: []}]", "operationId: caps, x-module: core, security: [{bearerAuth: [read]}]", 1)),
		Entry("x-scope on a public operation", strings.Replace(gateSpec, "operationId: open, x-module: core, security: [],", "operationId: open, x-module: core, x-scope: read, security: [],", 1)),
		Entry("x-scope that is not a string", strings.Replace(gateSpec, "x-scope: read", "x-scope: [read]", 1)),
		Entry("x-scope not in KnownScopes", strings.Replace(gateSpec, "x-module: password\n      x-scope: password", "x-module: admin\n      x-scope: admin", 1)),
	)

	DescribeTable("refuses rules that name an operation missing from the spec",
		func(set func(*gateRules) *map[string]bool) {
			doc, err := openapi3.NewLoader().LoadFromData([]byte(gateSpec))
			Expect(err).ToNot(HaveOccurred())
			rules := testGateRules
			m := set(&rules)
			*m = maps.Clone(*m)
			(*m)["typo"] = true
			_, err = newGate(doc, chi.NewRouter(), fa, rules)
			Expect(err).To(MatchError(ContainSubstring("typo")))
		},
		Entry("limited", func(r *gateRules) *map[string]bool { return &r.limited }),
		Entry("noScope", func(r *gateRules) *map[string]bool { return &r.noScope }),
		Entry("grantOps", func(r *gateRules) *map[string]bool { return &r.grantOps }),
		Entry("noStore", func(r *gateRules) *map[string]bool { return &r.noStore }),
	)

	It("checks routes against the spec in both directions", func() {
		Expect(g.checkRoutes()).To(Succeed())

		mux.Get("/extra", func(http.ResponseWriter, *http.Request) {})
		Expect(g.checkRoutes()).To(MatchError(ContainSubstring("GET /extra is not in the spec")))

		extraOp := strings.Replace(gateSpec, "components:", `  /unrouted:
    get: {operationId: unrouted, x-module: core, security: [], responses: {'200': {description: ok}}}
components:`, 1)
		_, err := build(extraOp)
		Expect(err).ToNot(HaveOccurred())
		Expect(g.checkRoutes()).To(MatchError(ContainSubstring("unrouted")))
	})
})

// captureLogs sends debug logs to a buffer for the rest of the spec.
func captureLogs() *bytes.Buffer {
	buf := &bytes.Buffer{}
	log.SetOutput(buf)
	log.SetLevel(log.LevelDebug)
	DeferCleanup(func() {
		log.SetOutput(os.Stderr)
		log.SetLevel(log.LevelFatal)
	})
	return buf
}
