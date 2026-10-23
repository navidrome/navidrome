package apiv1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/api"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAPIv1(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "API v1 Suite")
}

var specRouter routers.Router

// One database for the suite (db.Db() is a process-wide singleton); each spec clears users and grants.
var _ = BeforeSuite(func() {
	DeferCleanup(configtest.SetupConfig())
	conf.Server.DbPath = filepath.Join(GinkgoT().TempDir(), "apiv1.db") + "?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000"
	DeferCleanup(db.Init(GinkgoT().Context()))
	realDS = persistence.New(db.Db())

	doc, err := openapi3.NewLoader().LoadFromData(api.SpecJSON())
	Expect(err).ToNot(HaveOccurred())
	specRouter, err = gorillamux.NewRouter(doc)
	Expect(err).ToNot(HaveOccurred())
})

// serve routes req through h mounted at /api/v1 and asserts the response conforms to the spec.
func serve(h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	root := chi.NewRouter()
	root.Mount("/api/v1", h)
	w := httptest.NewRecorder()
	root.ServeHTTP(w, req)
	validateAgainstSpec(req, w)
	return w
}

// testClient drives a router end to end through serve, so every response is also checked against the spec.
type testClient struct {
	ctx    context.Context
	router http.Handler
}

func (c testClient) call(method, path, bearer string, body any) *httptest.ResponseRecorder {
	if body == nil {
		return c.callRaw(method, path, bearer, "")
	}
	b, _ := json.Marshal(body)
	return c.callRaw(method, path, bearer, string(b))
}

// callRaw sends body verbatim, for JSON a map cannot express, like keys differing only in case.
func (c testClient) callRaw(method, path, bearer, body string) *httptest.ResponseRecorder {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequestWithContext(c.ctx, method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequestWithContext(c.ctx, method, path, nil)
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	return serve(c.router, req)
}

func (c testClient) setup() GrantCreated {
	w := c.call(http.MethodPost, "/api/v1/auth/setup", "", creds("admin", "pw"))
	ExpectWithOffset(1, w.Code).To(Equal(http.StatusCreated), w.Body.String())
	var gc GrantCreated
	decodeJSON(w, &gc)
	return gc
}

// login signs in as the admin created by setup; nil scopes asks for all of them.
func (c testClient) login(scopes []string) GrantCreated {
	body := creds("admin", "pw")
	if scopes != nil {
		body["scopes"] = scopes
	}
	w := c.call(http.MethodPost, "/api/v1/auth/login", "", body)
	ExpectWithOffset(1, w.Code).To(Equal(http.StatusOK), w.Body.String())
	var gc GrantCreated
	decodeJSON(w, &gc)
	return gc
}

func creds(user, pw string) map[string]any {
	return map[string]any{"username": user, "password": pw, "client": "TestApp", "clientVersion": "1.0"}
}

func decodeJSON(w *httptest.ResponseRecorder, v any) {
	ExpectWithOffset(1, json.Unmarshal(w.Body.Bytes(), v)).To(Succeed(), w.Body.String())
}

func validateAgainstSpec(req *http.Request, w *httptest.ResponseRecorder) {
	route, pathParams, err := specRouter.FindRoute(req)
	if errors.Is(err, routers.ErrPathNotFound) || errors.Is(err, routers.ErrMethodNotAllowed) {
		return
	}
	ExpectWithOffset(2, err).ToNot(HaveOccurred())
	input := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request: req, PathParams: pathParams, Route: route,
		},
		Status:  w.Code,
		Header:  w.Header(),
		Body:    io.NopCloser(bytes.NewReader(w.Body.Bytes())),
		Options: &openapi3filter.Options{IncludeResponseStatus: true},
	}
	ExpectWithOffset(2, openapi3filter.ValidateResponse(req.Context(), input)).To(Succeed(),
		"response for %s %s does not conform to the spec", req.Method, req.URL.Path)
}
