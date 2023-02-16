// Package gg implements simple "extensions" to Go language. Based on https://github.com/icza/gog
package gg

// IfZero returns v if it is a non-zero value, orElse otherwise.
//
// This is similar to elvis operator (?:) in Groovy and other languages.
// Note: Different from the real elvis operator, the orElse expression will always get evaluated.
func IfZero[T comparable](v T, orElse T) T {
	var zero T
	if v != zero {
		return v
	}
	return orElse
}
