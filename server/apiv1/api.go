package apiv1

import (
	"errors"
	"net/http"
	"runtime/debug"

	"github.com/go-chi/chi/v5"
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

var _ StrictServerInterface = (*Router)(nil)

func (rt *Router) routes() http.Handler {
	r := chi.NewRouter()
	r.Use(problemRecoverer)
	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		writeProblemStatus(w, req, http.StatusNotFound, "not_found", "no such endpoint")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, req *http.Request) {
		writeProblemStatus(w, req, http.StatusMethodNotAllowed, "method_not_allowed", "")
	})

	r.Get("/openapi.json", rt.serveSpecJSON)
	r.Get("/openapi.yaml", rt.serveSpecYAML)

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
			writeProblemStatus(w, r, http.StatusInternalServerError, "internal", "")
		}()
		next.ServeHTTP(w, r)
	})
}
