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
	root := buildRouteTrie(r)
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		normalizeRequestPath(req, root)
		r.ServeHTTP(w, req)
	})
}

// routeNode is one level of the case-insensitive route trie, mirroring the shape of the actual
// routing tree. Literal children are keyed by their lower-cased segment and carry the case they
// were registered with; paramChild is the single merged subtree for whatever value fills a
// "{param}" segment at this position. Structuring this as a trie (rather than one flat
// segment->case map) matters because two unrelated routes can reuse the same segment name with
// different casing at different depths (e.g. "Info" in "/System/Info/Public" vs "info" in
// "/AudioMuseAI/info") — a flat map would let one overwrite the other's canonical case.
type routeNode struct {
	canonical  string
	children   map[string]*routeNode
	paramChild *routeNode
}

func newRouteNode() *routeNode {
	return &routeNode{children: map[string]*routeNode{}}
}

// buildRouteTrie walks every registered route into a routeNode trie keyed by path position.
func buildRouteTrie(router chi.Router) *routeNode {
	root := newRouteNode()
	_ = chi.Walk(router, func(_, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		node := root
		for seg := range strings.SplitSeq(route, "/") {
			if seg == "" {
				continue
			}
			if strings.Contains(seg, "{") {
				if node.paramChild == nil {
					node.paramChild = newRouteNode()
				}
				node = node.paramChild
				continue
			}
			key := strings.ToLower(seg)
			child, ok := node.children[key]
			if !ok {
				child = newRouteNode()
				node.children[key] = child
			}
			child.canonical = seg
			node = child
		}
		return nil
	})
	return root
}

// normalizeRequestPath rewrites literal path segments to the case routes were registered with.
// It must run before chi's matching. When the router is mounted under a parent, chi has already
// stripped the mount prefix and matches against RouteContext.RoutePath rather than r.URL.Path, so
// that's what must be normalized here.
func normalizeRequestPath(r *http.Request, root *routeNode) {
	if rctx := chi.RouteContext(r.Context()); rctx != nil && rctx.RoutePath != "" {
		rctx.RoutePath = normalizeCase(rctx.RoutePath, root)
		return
	}
	r.URL.Path = normalizeCase(r.URL.Path, root)
}

// normalizeCase rewrites each "/"-separated literal segment of path to the case it was
// registered with, descending the route trie in step so a segment name reused at a different
// position (e.g. "info") is judged against the right subtree instead of a global one. Once a
// segment falls outside the trie's literals (a param value, or a path with no matching route),
// it falls back to the position's paramChild so any later literal segments (e.g.
// ".../Images/{type}") still get normalized.
func normalizeCase(path string, root *routeNode) string {
	segs := strings.Split(path, "/")
	node := root
	for i, seg := range segs {
		if seg == "" || node == nil {
			continue
		}
		if child, ok := node.children[strings.ToLower(seg)]; ok {
			segs[i] = child.canonical
			node = child
			continue
		}
		// A segment like "STREAM.mp3" comes from a mixed literal+param route (e.g.
		// "stream.{container}"), whose literal prefix ("stream") is registered separately:
		// normalize that prefix and lower-case the extension so chi's case-sensitive match hits.
		if prefix, suffix, found := strings.Cut(seg, "."); found {
			if child, ok := node.children[strings.ToLower(prefix)]; ok {
				segs[i] = child.canonical + "." + strings.ToLower(suffix)
				node = child
				continue
			}
		}
		node = node.paramChild
	}
	return strings.Join(segs, "/")
}
