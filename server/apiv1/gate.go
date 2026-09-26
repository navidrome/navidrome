package apiv1

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/apiauth"
	"github.com/navidrome/navidrome/log"
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
	route   *routers.Route
	kind    authKind
	scope   string
	limited bool
	noStore bool
}

func (o *gateOp) id() string { return o.route.Operation.OperationID }

type opKey struct{ method, path string }

type gate struct {
	mux     chi.Routes
	ops     map[opKey]*gateOp
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
	noStore  map[string]bool // operations whose responses carry a secret or token
}

func newGate(doc *openapi3.T, mux chi.Routes, auth authenticator, rules gateRules) (*gate, error) {
	g := &gate{mux: mux, ops: map[opKey]*gateOp{}, auth: auth}
	ids := map[string]bool{}
	for path, item := range doc.Paths.Map() {
		for method, op := range item.Operations() {
			gop, err := buildGateOp(doc, path, item, method, op, rules)
			if err != nil {
				return nil, err
			}
			g.ops[opKey{method, path}] = gop
			ids[op.OperationID] = true
		}
	}
	if err := rules.check(ids); err != nil {
		return nil, err
	}
	if conf.Server.AuthRequestLimit > 0 {
		g.limiter = server.ClientIPRateLimiter(conf.Server.AuthRequestLimit, conf.Server.AuthWindowLength,
			httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
				writeProblemStatus(w, r, http.StatusTooManyRequests, ProblemCodeRateLimited, "too many requests")
			}))
	}
	return g, nil
}

// check fails on a rule naming an operation the spec lacks, so a typo cannot silently disable the rule.
func (rules gateRules) check(ids map[string]bool) error {
	sets := map[string]map[string]bool{"limited": rules.limited, "noScope": rules.noScope, "grantOps": rules.grantOps, "noStore": rules.noStore}
	for name, set := range sets {
		for id := range set {
			if !ids[id] {
				return fmt.Errorf("gate rule %s names unknown operation %s", name, id)
			}
		}
	}
	return nil
}

// buildGateOp enforces the allowed security forms, so a spec edit cannot silently drop a requirement.
func buildGateOp(doc *openapi3.T, path string, item *openapi3.PathItem, method string, op *openapi3.Operation, rules gateRules) (*gateOp, error) {
	id := op.OperationID
	gop := &gateOp{
		route:   &routers.Route{Spec: doc, Path: path, PathItem: item, Method: method, Operation: op},
		limited: rules.limited[id],
		noStore: rules.noStore[id],
	}
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
		base := cmp.Or(moduleScope[module], module)
		if scope != base && scope != base+":write" {
			return nil, fmt.Errorf("operation %s: x-scope %q does not match module %q", id, scope, module)
		}
		if !slices.Contains(apiauth.KnownScopes, scope) {
			return nil, fmt.Errorf("operation %s: unknown x-scope %q", id, scope)
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
		if _, ok := g.ops[opKey{method, route}]; !ok {
			return fmt.Errorf("route %s %s is not in the spec", method, route)
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, op := range g.ops {
		if g.mux.Find(chi.NewRouteContext(), op.route.Method, op.route.Path) != op.route.Path {
			return fmt.Errorf("spec operation %s (%s %s) has no route", op.id(), op.route.Method, op.route.Path)
		}
	}
	return nil
}

func (g *gate) handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := routeMethod(r)
		rctx := chi.NewRouteContext()
		pattern := g.mux.Find(rctx, method, routePath(r))
		if pattern == "" {
			next.ServeHTTP(w, r)
			return
		}
		op, ok := g.ops[opKey{method, pattern}]
		if !ok {
			log.Error(r.Context(), "API v1: routed pattern missing from the spec", "method", method, "pattern", pattern)
			writeProblemStatus(w, r, http.StatusInternalServerError, ProblemCodeInternal, "")
			return
		}
		if op.noStore {
			w.Header().Set("Cache-Control", "no-store")
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
		writeProblemStatus(w, r, http.StatusUnauthorized, ProblemCodeUnauthorized, "")
		return r, false
	}
	ip := server.ClientAddr(r)
	var p *apiauth.Principal
	var err error
	if op.kind == authGrant {
		p, err = g.auth.ResolveGrant(r.Context(), token, ip)
	} else {
		p, err = g.auth.Authenticate(r.Context(), token, ip)
	}
	if err == nil && op.scope != "" && !apiauth.Satisfies(p.Scopes, op.scope) {
		err = apiauth.ErrInsufficientScope
	}
	if errors.Is(err, apiauth.ErrInsufficientScope) {
		err = &scopeError{scope: op.scope}
	}
	if err != nil {
		writeProblem(w, r, err)
		return r, false
	}
	ctx := apiauth.WithPrincipal(request.WithUser(r.Context(), p.User), p)
	return r.WithContext(ctx), true
}

func bearerToken(r *http.Request) (string, bool) {
	scheme, token, ok := strings.Cut(strings.TrimSpace(r.Header.Get("Authorization")), " ")
	token = strings.TrimSpace(token)
	if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return "", false
	}
	return token, true
}

var validationOptions = &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc, MultiError: true}

func (g *gate) validate(w http.ResponseWriter, r *http.Request, op *gateOp, rctx *chi.Context) bool {
	params := make(map[string]string, len(rctx.URLParams.Keys))
	for i, k := range rctx.URLParams.Keys {
		params[k] = rctx.URLParams.Values[i]
	}
	err := openapi3filter.ValidateRequest(r.Context(), &openapi3filter.RequestValidationInput{
		Request: r, PathParams: params, Route: op.route, Options: validationOptions,
	})
	if tooLarge(err) {
		writeProblem(w, r, ClientError(err, tooLargeDetail))
		return false
	}
	var fields []ValidationError
	if err != nil {
		fields = sanitizeValidation(err)
	} else if fields = jsonBodyFields(r, op.route.Operation); len(fields) == 0 {
		return true
	}
	log.Debug(r.Context(), "API v1: request failed validation", "operation", op.id(), "errors", fields)
	writeProblemStatus(w, r, http.StatusBadRequest, ProblemCodeValidation, "the request does not match the API schema", fields...)
	return false
}

// jsonBodyFields checks what kin-openapi misses in a JSON body: data after the first value, which Go's decoder
// ignores, and keys that only case-fold to a declared property, which encoding/json decodes into that property.
func jsonBodyFields(r *http.Request, op *openapi3.Operation) []ValidationError {
	if op.RequestBody == nil || op.RequestBody.Value == nil || r.Body == nil {
		return nil
	}
	// Keyed on the spec, not the request's Content-Type: the handlers decode JSON whatever the header says.
	media := op.RequestBody.Value.Content.Get("application/json")
	if media == nil || media.Schema == nil {
		return nil
	}
	data, err := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(data))
	if err != nil || len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	var body any
	if err := dec.Decode(&body); err != nil {
		return []ValidationError{{Field: "", Message: "must be a single JSON value"}}
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return []ValidationError{{Field: "", Message: "must be a single JSON value"}}
	}
	var out []ValidationError
	collectCaseAliases(body, media.Schema.Value, "", &out)
	return out
}

func collectCaseAliases(v any, schema *openapi3.Schema, path string, out *[]ValidationError) {
	if schema == nil {
		return
	}
	switch v := v.(type) {
	case map[string]any:
		for _, key := range slices.Sorted(maps.Keys(v)) {
			field := joinField(path, key)
			if prop, ok := schema.Properties[key]; ok {
				if prop != nil {
					collectCaseAliases(v[key], prop.Value, field, out)
				}
				continue
			}
			for name := range schema.Properties {
				if strings.EqualFold(key, name) {
					*out = append(*out, ValidationError{Field: field, Message: "must match the field name exactly"})
					break
				}
			}
		}
	case []any:
		if schema.Items == nil {
			return
		}
		for i, item := range v {
			collectCaseAliases(item, schema.Items.Value, joinField(path, strconv.Itoa(i)), out)
		}
	}
}

func joinField(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
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
