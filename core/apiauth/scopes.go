package apiauth

import (
	"slices"
	"strings"
)

const (
	ScopeAll      = "all"
	ScopeRead     = "read"
	ScopePassword = "password"
	ScopeAdmin    = "admin"
)

// KnownScopes lists the scopes of modules this server implements; `all` expands to these.
var KnownScopes = []string{ScopeRead, ScopePassword}

func known(s string) bool {
	return slices.Contains(KnownScopes, s)
}

func grantable(s string, isAdmin bool) bool {
	return known(s) && (s != ScopeAdmin || isAdmin)
}

func normalize(in []string) []string {
	out := slices.Clone(in)
	slices.Sort(out)
	return slices.Compact(out)
}

// Entitled returns the scopes a new grant stores.
func Entitled(requested []string, isAdmin bool) []string {
	if requested == nil {
		return []string{ScopeAll}
	}
	var out []string
	for _, s := range requested {
		if s == ScopeAll || grantable(s, isAdmin) {
			out = append(out, s)
		}
	}
	return normalize(out)
}

// Expand turns a grant's stored scopes into the concrete scopes it carries right now.
func Expand(granted []string, isAdmin bool) []string {
	var out []string
	for _, s := range granted {
		if s == ScopeAll {
			out = append(out, KnownScopes...)
			continue
		}
		out = append(out, s)
	}
	return Allowed(out, isAdmin)
}

// Allowed keeps the concrete scopes the user may hold now; unlike Expand it never widens `all`.
func Allowed(scopes []string, isAdmin bool) []string {
	out := slices.DeleteFunc(slices.Clone(scopes), func(s string) bool { return !grantable(s, isAdmin) })
	return normalize(out)
}

func Satisfies(scopes []string, required string) bool {
	return slices.Contains(scopes, required) ||
		(!strings.HasSuffix(required, ":write") && slices.Contains(scopes, required+":write"))
}
