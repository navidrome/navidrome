package apiv1

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/apiauth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server"
)

type authenticator interface {
	Authenticate(ctx context.Context, token, ip string) (*apiauth.Principal, error)
	ResolveGrant(ctx context.Context, secret, ip string) (*apiauth.Principal, error)
}

type authKind int

const (
	authPublic authKind = iota
	authToken
	authGrant
)

type gateOp struct {
	id      string
	route   *routers.Route
	kind    authKind
	scope   string
	limited bool
}

type gate struct {
	mux     chi.Routes
	ops     map[string]*gateOp
	auth    authenticator
	limiter func(http.Handler) http.Handler
}

// Modules that ride another module's scope; every other module's scope is its own name.
var moduleScope = map[string]string{
	"core":            apiauth.ScopeRead,
	"transcoding":     "streaming",
	"custom-tags":     apiauth.ScopeRead,
	"grouping":        apiauth.ScopeRead,
	"smart-playlists": "playlists:write",
}

type gateRules struct {
	limited  map[string]bool // login-type operations, throttled per client IP
	noScope  map[string]bool // the only token operations allowed without x-scope
	grantOps map[string]bool // the only operations allowed to use grantAuth
}

func newGate(doc *openapi3.T, mux chi.Routes, auth authenticator, rules gateRules) (*gate, error) {
	g := &gate{mux: mux, ops: map[string]*gateOp{}, auth: auth}
	for path, item := range doc.Paths.Map() {
		for method, op := range item.Operations() {
			gop, err := buildGateOp(doc, path, item, method, op, rules)
			if err != nil {
				return nil, err
			}
			gop.limited = rules.limited[op.OperationID]
			g.ops[method+" "+path] = gop
		}
	}
	if conf.Server.AuthRequestLimit > 0 {
		g.limiter = httprate.LimitBy(conf.Server.AuthRequestLimit, conf.Server.AuthWindowLength,
			func(r *http.Request) (string, error) { return server.ClientIP(r), nil },
			httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
				writeProblemStatus(w, r, http.StatusTooManyRequests, ProblemCodeRateLimited, "too many requests")
			}))
	}
	return g, nil
}

// buildGateOp enforces the allowed security forms, so a spec edit cannot silently drop a requirement.
func buildGateOp(doc *openapi3.T, path string, item *openapi3.PathItem, method string, op *openapi3.Operation, rules gateRules) (*gateOp, error) {
	id := op.OperationID
	gop := &gateOp{id: id, route: &routers.Route{Spec: doc, Path: path, PathItem: item, Method: method, Operation: op}}
	if op.Security == nil {
		return nil, fmt.Errorf("operation %s must declare security explicitly", id)
	}
	rawScope, hasScope := op.Extensions["x-scope"]
	scope, isString := rawScope.(string)
	if hasScope && (!isString || scope == "") {
		return nil, fmt.Errorf("operation %s: x-scope must be a non-empty string", id)
	}
	module, _ := op.Extensions["x-module"].(string)
	switch reqs := *op.Security; {
	case len(reqs) == 0:
		gop.kind = authPublic
	case len(reqs) == 1 && isScheme(reqs[0], "bearerAuth"):
		gop.kind = authToken
	case len(reqs) == 1 && isScheme(reqs[0], "grantAuth") && rules.grantOps[id]:
		gop.kind = authGrant
	default:
		return nil, fmt.Errorf("operation %s has a security requirement outside the allowed forms", id)
	}
	if gop.kind == authToken && scope == "" && !rules.noScope[id] {
		return nil, fmt.Errorf("operation %s: bearerAuth needs x-scope", id)
	}
	if scope != "" {
		if gop.kind != authToken {
			return nil, fmt.Errorf("operation %s: x-scope needs bearerAuth", id)
		}
		base := module
		if s, ok := moduleScope[module]; ok {
			base = s
		}
		if scope != base && scope != base+":write" {
			return nil, fmt.Errorf("operation %s: x-scope %q does not match module %q", op.OperationID, scope, module)
		}
		if !slices.Contains(apiauth.KnownScopes, scope) && scope != apiauth.ScopeAdmin {
			return nil, fmt.Errorf("operation %s: unknown x-scope %q", op.OperationID, scope)
		}
	}
	gop.scope = scope
	return gop, nil
}

// isScheme requires the scheme alone with an empty scope list, as OpenAPI 3.0.3 demands for http schemes.
func isScheme(req openapi3.SecurityRequirement, name string) bool {
	scopes, ok := req[name]
	return ok && len(req) == 1 && len(scopes) == 0
}

// checkRoutes fails when a routed pattern has no spec operation or a spec operation has no route.
func (g *gate) checkRoutes() error {
	err := chi.Walk(g.mux, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if _, ok := g.ops[method+" "+route]; !ok {
			return fmt.Errorf("route %s %s is not in the spec", method, route)
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, op := range g.ops {
		if g.mux.Find(chi.NewRouteContext(), op.route.Method, op.route.Path) != op.route.Path {
			return fmt.Errorf("spec operation %s (%s %s) has no route", op.id, op.route.Method, op.route.Path)
		}
	}
	return nil
}

func (g *gate) handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := routePath(r)
		method := r.Method
		if method == http.MethodHead && !g.mux.Match(chi.NewRouteContext(), http.MethodHead, path) {
			method = http.MethodGet
		}
		rctx := chi.NewRouteContext()
		pattern := g.mux.Find(rctx, method, path)
		if pattern == "" {
			next.ServeHTTP(w, r)
			return
		}
		op, ok := g.ops[method+" "+pattern]
		if !ok {
			log.Error(r.Context(), "API v1: routed pattern missing from the spec", "method", method, "pattern", pattern)
			writeProblemStatus(w, r, http.StatusInternalServerError, ProblemCodeInternal, "")
			return
		}
		serve := func(w http.ResponseWriter, r *http.Request) {
			r, ok := g.authorize(w, r, op)
			if !ok {
				return
			}
			if !g.validate(w, r, op, rctx) {
				return
			}
			next.ServeHTTP(w, r)
		}
		if op.limited && g.limiter != nil {
			g.limiter(http.HandlerFunc(serve)).ServeHTTP(w, r)
			return
		}
		serve(w, r)
	})
}

func (g *gate) authorize(w http.ResponseWriter, r *http.Request, op *gateOp) (*http.Request, bool) {
	if op.kind == authPublic {
		return r, true
	}
	token, ok := bearerToken(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", "Bearer")
		writeProblemStatus(w, r, http.StatusUnauthorized, ProblemCodeUnauthorized, "")
		return r, false
	}
	ip := server.ClientIP(r)
	var p *apiauth.Principal
	var err error
	if op.kind == authGrant {
		p, err = g.auth.ResolveGrant(r.Context(), token, ip)
	} else {
		p, err = g.auth.Authenticate(r.Context(), token, ip)
	}
	switch {
	case errors.Is(err, apiauth.ErrTokenExpired), errors.Is(err, model.ErrInvalidAuth):
		w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
		writeProblem(w, r, err)
		return r, false
	case errors.Is(err, apiauth.ErrInsufficientScope):
		insufficientScope(w, r, op, err)
		return r, false
	case err != nil:
		writeProblem(w, r, err)
		return r, false
	}
	if op.scope != "" && !apiauth.Satisfies(p.Scopes, op.scope) {
		insufficientScope(w, r, op, apiauth.ErrInsufficientScope)
		return r, false
	}
	ctx := apiauth.WithPrincipal(request.WithUser(r.Context(), p.User), p)
	return r.WithContext(ctx), true
}

func insufficientScope(w http.ResponseWriter, r *http.Request, op *gateOp, err error) {
	challenge := `Bearer error="insufficient_scope"`
	if op.scope != "" {
		challenge += fmt.Sprintf(`, scope=%q`, op.scope)
	}
	w.Header().Set("WWW-Authenticate", challenge)
	writeProblem(w, r, err)
}

func bearerToken(r *http.Request) (string, bool) {
	scheme, token, ok := strings.Cut(strings.TrimSpace(r.Header.Get("Authorization")), " ")
	token = strings.TrimSpace(token)
	if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return "", false
	}
	return token, true
}

func (g *gate) validate(w http.ResponseWriter, r *http.Request, op *gateOp, rctx *chi.Context) bool {
	params := map[string]string{}
	for i, k := range rctx.URLParams.Keys {
		params[k] = rctx.URLParams.Values[i]
	}
	err := openapi3filter.ValidateRequest(r.Context(), &openapi3filter.RequestValidationInput{
		Request: r, PathParams: params, Route: op.route,
		Options: &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc, MultiError: true},
	})
	if err == nil {
		return true
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeProblemStatus(w, r, http.StatusRequestEntityTooLarge, ProblemCodePayloadTooLarge, "request body too large")
		return false
	}
	fields := sanitizeValidation(err)
	log.Debug(r.Context(), "API v1: request failed validation", "operation", op.id, "errors", fields)
	writeProblemStatus(w, r, http.StatusBadRequest, ProblemCodeValidation, "the request does not match the API schema", fields...)
	return false
}

var missingProperty = regexp.MustCompile(`property "([^"]+)" is missing`)

// sanitizeValidation keeps only field paths and fixed messages: kin-openapi errors can embed the submitted value.
// It walks wrappers by concrete type, not errors.As, because MultiError.As would skip the RequestError that names the parameter.
func sanitizeValidation(err error) []ValidationError {
	var out []ValidationError
	var walk func(err error, param string)
	walk = func(err error, param string) {
		switch e := err.(type) { //nolint:errorlint
		case openapi3.MultiError:
			for _, child := range e {
				walk(child, param)
			}
		case *openapi3filter.RequestError:
			if e.Parameter != nil {
				param = e.Parameter.Name
			}
			switch {
			case errors.Is(e.Err, openapi3filter.ErrInvalidRequired), errors.Is(e.Err, openapi3filter.ErrInvalidEmptyValue):
				out = append(out, ValidationError{Field: param, Message: "is required"})
			case e.Err != nil:
				walk(e.Err, param)
			default:
				out = append(out, ValidationError{Field: param, Message: "is invalid"})
			}
		case *openapi3.SchemaError:
			field := strings.Join(e.JSONPointer(), ".")
			if field == "" && e.SchemaField == "required" {
				if m := missingProperty.FindStringSubmatch(e.Reason); m != nil {
					field = m[1]
				}
			}
			switch {
			case param != "" && field != "":
				field = param + "." + field
			case field == "":
				field = param
			}
			out = append(out, ValidationError{Field: field, Message: schemaMessage(e.SchemaField)})
		default:
			if inner := errors.Unwrap(err); inner != nil {
				walk(inner, param)
				return
			}
			out = append(out, ValidationError{Field: param, Message: "is invalid"})
		}
	}
	walk(err, "")
	return out
}

func schemaMessage(keyword string) string {
	switch keyword {
	case "required":
		return "is required"
	case "maxLength", "maxItems":
		return "is too long"
	case "minLength", "minItems":
		return "is too short"
	case "maximum", "exclusiveMaximum":
		return "is too large"
	case "minimum", "exclusiveMinimum":
		return "is too small"
	case "pattern", "format":
		return "has an invalid format"
	case "enum":
		return "is not an allowed value"
	case "type", "nullable":
		return "has the wrong type"
	default:
		return "is invalid"
	}
}
