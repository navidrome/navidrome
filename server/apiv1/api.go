package apiv1

import (
	"cmp"
	"errors"
	"net/http"
	"runtime/debug"
	"slices"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/api"
	"github.com/navidrome/navidrome/core/apiauth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

const maxBodyBytes = 1 << 20

type Router struct {
	http.Handler
	ds   model.DataStore
	auth *apiauth.Service
}

func New(ds model.DataStore) *Router {
	rt := &Router{ds: ds, auth: apiauth.New(ds)}
	rt.Handler = rt.routes()
	return rt
}

var gateRulesV1 = gateRules{
	limited:  map[string]bool{"login": true, "setupFirstAdmin": true, "changePassword": true},
	noScope:  map[string]bool{"getCapabilities": true},
	grantOps: map[string]bool{"createAccessToken": true},
}

func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		}
		next.ServeHTTP(w, r)
	})
}

func (rt *Router) routes() http.Handler {
	r := chi.NewRouter()
	doc, err := openapi3.NewLoader().LoadFromData(api.SpecJSON())
	if err != nil {
		log.Fatal("API v1: cannot load the embedded OpenAPI spec", err)
	}
	g, err := newGate(doc, r, rt.auth, gateRulesV1)
	if err != nil {
		log.Fatal("API v1: the embedded OpenAPI spec breaks the security rules", err)
	}
	r.Use(referenceIDMiddleware, problemRecoverer, headAsGet(r), limitBody, g.handler)
	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		writeProblemStatus(w, req, http.StatusNotFound, ProblemCodeNotFound, "no such endpoint")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Allow", strings.Join(allowedMethods(r, req), ", "))
		writeProblemStatus(w, req, http.StatusMethodNotAllowed, ProblemCodeMethodNotAllowed, "")
	})

	r.Get("/openapi.json", specHandler(withBasePath(api.SpecJSON(), `"url": `, true), "application/json"))
	r.Get("/openapi.yaml", specHandler(withBasePath(api.SpecYAML(), "url: ", false), "application/yaml"))

	strict := NewStrictHandlerWithOptions(rt, nil, StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, req *http.Request, err error) {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				writeProblemStatus(w, req, http.StatusRequestEntityTooLarge, ProblemCodePayloadTooLarge, "request body too large")
				return
			}
			writeProblemStatus(w, req, http.StatusBadRequest, ProblemCodeValidation, "request body is not valid JSON")
		},
		ResponseErrorHandlerFunc: writeProblem,
	})
	HandlerWithOptions(strict, ChiServerOptions{BaseRouter: r, ErrorHandlerFunc: bindingErrorHandler})
	if err := g.checkRoutes(); err != nil {
		log.Fatal("API v1: routes and the embedded OpenAPI spec disagree", err)
	}
	return r
}

func problemRecoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(rec)
			}
			log.Error(r.Context(), "API v1: panic in handler", "panic", rec, "stack", string(debug.Stack()))
			writeProblemStatus(w, r, http.StatusInternalServerError, ProblemCodeInternal, "")
		}()
		next.ServeHTTP(w, r)
	})
}

var routableMethods = []string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}

// Looks routes up on mux itself: chi's RouteContext().Routes points at the parent router when mounted.
func allowedMethods(mux chi.Routes, req *http.Request) []string {
	path := routePath(req)
	var allowed []string
	for _, m := range routableMethods {
		if mux.Match(chi.NewRouteContext(), m, path) || (m == http.MethodHead && slices.Contains(allowed, http.MethodGet)) {
			allowed = append(allowed, m)
		}
	}
	return allowed
}

func headAsGet(mux chi.Routes) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodHead && !mux.Match(chi.NewRouteContext(), http.MethodHead, routePath(req)) {
				chi.RouteContext(req.Context()).RouteMethod = http.MethodGet
			}
			next.ServeHTTP(w, req)
		})
	}
}

// routePath must pick the same path chi's routeHTTP dispatches on, or the gate could vet a different route.
func routePath(req *http.Request) string {
	if rctx := chi.RouteContext(req.Context()); rctx != nil && rctx.RoutePath != "" {
		return rctx.RoutePath
	}
	return cmp.Or(req.URL.RawPath, req.URL.Path, "/")
}
