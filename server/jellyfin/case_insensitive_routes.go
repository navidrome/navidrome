package jellyfin

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// caseInsensitivePaths normalizes each request path's literal segments to the case they were
// registered with before delegating to r, since Jellyfin clients route case-insensitively but
// chi matches case-sensitively. Param placeholders (e.g. "{itemId}") aren't literals, so id
// segments pass through untouched.
func caseInsensitivePaths(r chi.Router) http.Handler {
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
// A segment like "STREAM.mp3" comes from a mixed literal+param route (e.g. "stream.{container}"),
// whose literal prefix ("stream") is registered separately: normalize that prefix and lower-case
// the extension so chi's case-sensitive match still hits.
func normalizeCase(path string, canon map[string]string) string {
	segs := strings.Split(path, "/")
	for i, seg := range segs {
		if canonical, ok := canon[strings.ToLower(seg)]; ok {
			segs[i] = canonical
		} else if prefix, suffix, found := strings.Cut(seg, "."); found {
			if canonical, ok := canon[strings.ToLower(prefix)]; ok {
				segs[i] = canonical + "." + strings.ToLower(suffix)
			}
		}
	}
	return strings.Join(segs, "/")
}
