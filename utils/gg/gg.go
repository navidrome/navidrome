// Package gg implements simple "extensions" to Go language. Based on https://github.com/icza/gog
package gg

// If returns v if it is a non-zero value, orElse otherwise.
//
// This is similar to elvis operator (?:) in Groovy and other languages.
// Note: Different from the real elvis operator, the orElse expression will always get evaluated.
func If[T comparable](v T, orElse T) T {
	var zero T
	if v != zero {
		return v
	}
	return orElse
}

// P returns a pointer to the input value
func P[T any](v T) *T {
	return &v
}

// V returns the value of the input pointer, or a zero value if the input pointer is nil.
func V[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}
