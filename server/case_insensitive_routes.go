package server

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// CaseInsensitivePaths wraps r with a handler that normalizes each request path's literal
// segments to the case they were registered with, then delegates to r. This is needed because
// some clients (e.g. real Jellyfin clients) route case-insensitively while chi's default matching
// is case-sensitive. Chi param placeholders (e.g. "{itemId}") are never treated as literals, so
// id segments always pass through untouched.
func CaseInsensitivePaths(r chi.Router) http.Handler {
	canon := canonicalRouteSegments(r)
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		normalizeRequestPath(req, canon)
		r.ServeHTTP(w, req)
	})
}

// canonicalRouteSegments walks every registered route and records, for each literal (non-param)
// "/"-separated segment, the case it was registered with, keyed by its lower-cased form (e.g.
// "audio" -> "Audio").
func canonicalRouteSegments(router chi.Router) map[string]string {
	canon := map[string]string{}
	_ = chi.Walk(router, func(_, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		for seg := range strings.SplitSeq(route, "/") {
			if seg == "" || strings.Contains(seg, "{") {
				continue
			}
			canon[strings.ToLower(seg)] = seg
		}
		return nil
	})
	return canon
}

// normalizeRequestPath rewrites literal path segments to the case routes were registered with.
// It must run before chi's matching. When the router is mounted under a parent, chi has already
// stripped the mount prefix and matches against RouteContext.RoutePath rather than r.URL.Path, so
// that's what must be normalized here.
func normalizeRequestPath(r *http.Request, canon map[string]string) {
	if rctx := chi.RouteContext(r.Context()); rctx != nil && rctx.RoutePath != "" {
		rctx.RoutePath = normalizeCase(rctx.RoutePath, canon)
		return
	}
	r.URL.Path = normalizeCase(r.URL.Path, canon)
}

// normalizeCase rewrites each "/"-separated literal segment of path to the case it was
// registered with in canon. Segments with no match (e.g. case-sensitive ids) are left untouched.
func normalizeCase(path string, canon map[string]string) string {
	segs := strings.Split(path, "/")
	for i, seg := range segs {
		if canonical, ok := canon[strings.ToLower(seg)]; ok {
			segs[i] = canonical
		}
	}
	return strings.Join(segs, "/")
}
