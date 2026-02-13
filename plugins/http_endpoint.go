package plugins

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/plugins/capabilities"
)

const maxEndpointBodySize = 1 << 20 // 1MB

// SubsonicAuthValidator validates Subsonic authentication and returns the user.
// This is set by the cmd/ package to avoid import cycles (plugins -> server/subsonic).
type SubsonicAuthValidator func(ds model.DataStore, r *http.Request) (*model.User, error)

// NativeAuthMiddleware is an HTTP middleware that authenticates using JWT tokens.
// This is set by the cmd/ package to avoid import cycles (plugins -> server).
type NativeAuthMiddleware func(ds model.DataStore) func(next http.Handler) http.Handler

// NewEndpointRouter creates an HTTP handler that dispatches requests to plugin endpoints.
// It should be mounted at both /ext and /rest/ext. The handler uses a catch-all pattern
// because Chi does not support adding routes after startup, and plugins can be loaded/unloaded
// at runtime. Plugin lookup happens per-request under RLock.
func NewEndpointRouter(manager *Manager, ds model.DataStore, subsonicAuth SubsonicAuthValidator, nativeAuth NativeAuthMiddleware) http.Handler {
	r := chi.NewRouter()

	// Apply rate limiting if configured
	if conf.Server.Plugins.EndpointRequestLimit > 0 {
		r.Use(httprate.LimitByIP(conf.Server.Plugins.EndpointRequestLimit, conf.Server.Plugins.EndpointRequestWindow))
	}

	h := &endpointHandler{
		manager:      manager,
		ds:           ds,
		subsonicAuth: subsonicAuth,
		nativeAuth:   nativeAuth,
	}
	r.HandleFunc("/{pluginID}/*", h.ServeHTTP)
	r.HandleFunc("/{pluginID}", h.ServeHTTP)
	return r
}

type endpointHandler struct {
	manager      *Manager
	ds           model.DataStore
	subsonicAuth SubsonicAuthValidator
	nativeAuth   NativeAuthMiddleware
}

func (h *endpointHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pluginID := chi.URLParam(r, "pluginID")

	h.manager.mu.RLock()
	p, ok := h.manager.plugins[pluginID]
	h.manager.mu.RUnlock()

	if !ok || !hasCapability(p.capabilities, CapabilityHTTPEndpoint) {
		http.NotFound(w, r)
		return
	}

	if p.manifest.Permissions == nil || p.manifest.Permissions.Endpoints == nil {
		http.NotFound(w, r)
		return
	}

	authType := p.manifest.Permissions.Endpoints.Auth

	switch authType {
	case EndpointsPermissionAuthSubsonic:
		h.serveWithSubsonicAuth(w, r, p)
	case EndpointsPermissionAuthNative:
		h.serveWithNativeAuth(w, r, p)
	case EndpointsPermissionAuthNone:
		h.dispatch(w, r, p)
	default:
		http.Error(w, "Unknown auth type", http.StatusInternalServerError)
	}
}

func (h *endpointHandler) serveWithSubsonicAuth(w http.ResponseWriter, r *http.Request, p *plugin) {
	usr, err := h.subsonicAuth(h.ds, r)
	if err != nil {
		log.Warn(r.Context(), "Plugin endpoint auth failed", "plugin", p.name, "auth", "subsonic", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	ctx := request.WithUser(r.Context(), *usr)
	h.dispatch(w, r.WithContext(ctx), p)
}

func (h *endpointHandler) serveWithNativeAuth(w http.ResponseWriter, r *http.Request, p *plugin) {
	h.nativeAuth(h.ds)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.dispatch(w, r, p)
	})).ServeHTTP(w, r)
}

func (h *endpointHandler) dispatch(w http.ResponseWriter, r *http.Request, p *plugin) {
	ctx := r.Context()

	// Check user authorization (skip for auth:"none")
	if p.manifest.Permissions.Endpoints.Auth != EndpointsPermissionAuthNone {
		user, ok := request.UserFrom(ctx)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if !p.userAccess.IsAllowed(user.ID) {
			log.Warn(ctx, "Plugin endpoint access denied", "plugin", p.name, "user", user.UserName)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	// Read request body with size limit
	body, err := io.ReadAll(io.LimitReader(r.Body, maxEndpointBodySize))
	if err != nil {
		log.Error(ctx, "Failed to read request body", "plugin", p.name, err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Build the plugin request
	// Normalize path: both /ext/plugin and /ext/plugin/ map to ""
	relPath := "/" + chi.URLParam(r, "*")
	if relPath == "/" || relPath == "" {
		relPath = ""
	}

	var httpUser *capabilities.HTTPUser
	if p.manifest.Permissions.Endpoints.Auth != EndpointsPermissionAuthNone {
		if user, ok := request.UserFrom(ctx); ok {
			httpUser = &capabilities.HTTPUser{
				ID:       user.ID,
				Username: user.UserName,
				Name:     user.Name,
				IsAdmin:  user.IsAdmin,
			}
		}
	}

	pluginReq := capabilities.HTTPHandleRequest{
		Method:  r.Method,
		Path:    relPath,
		Query:   r.URL.RawQuery,
		Headers: r.Header,
		Body:    body,
		User:    httpUser,
	}

	// Call the plugin using binary framing for []byte Body fields
	resp, err := callPluginFunctionRaw(
		ctx, p, FuncHTTPHandleRequest,
		pluginReq, pluginReq.Body,
		func(r *capabilities.HTTPHandleResponse, raw []byte) { r.Body = raw },
	)
	if err != nil {
		log.Error(ctx, "Plugin endpoint call failed", "plugin", p.name, "path", relPath, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Write response headers from plugin
	for key, values := range resp.Headers {
		for _, v := range values {
			w.Header().Add(key, v)
		}
	}

	// Security hardening: override any plugin-set security headers
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; img-src data:; sandbox")

	// Write status code (default to 200)
	status := int(resp.Status)
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)

	// Write response body
	if len(resp.Body) > 0 {
		if _, err := w.Write(resp.Body); err != nil {
			log.Error(ctx, "Failed to write plugin endpoint response", "plugin", p.name, err)
		}
	}
}
