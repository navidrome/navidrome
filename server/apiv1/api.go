package apiv1

import (
	"errors"
	"net/http"
	"runtime/debug"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/api"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type Router struct {
	http.Handler
	ds model.DataStore
}

func New(ds model.DataStore) *Router {
	rt := &Router{ds: ds}
	rt.Handler = rt.routes()
	return rt
}

func (rt *Router) routes() http.Handler {
	r := chi.NewRouter()
	r.Use(problemRecoverer, headAsGet(r))
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
			writeProblemStatus(w, req, http.StatusBadRequest, "validation", err.Error())
		},
		ResponseErrorHandlerFunc: writeProblem,
	})
	HandlerWithOptions(strict, ChiServerOptions{BaseRouter: r, ErrorHandlerFunc: bindingErrorHandler})
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

func routePath(req *http.Request) string {
	if rctx := chi.RouteContext(req.Context()); rctx != nil && rctx.RoutePath != "" {
		return rctx.RoutePath
	}
	return req.URL.Path
}
