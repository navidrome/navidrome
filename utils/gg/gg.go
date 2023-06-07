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

// P returns a pointer to a value, or nil if the value is the zero value of the type.
func P[T comparable](t T) *T {
	var zero T
	if t == zero {
		return nil
	}
	return &t
}

// V returns the value of a pointer, or the zero value of the type if the pointer is nil.
func V[T comparable](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}

// Coalesce returns the first non-zero value from listed arguments.
// Returns the zero value of the type parameter if no arguments are given or all are the zero value.
// Useful when you want to initialize a variable to the first non-zero value from a list of fallback values.
//
// For example:
//
//	hostVal := Coalesce(hostName, os.Getenv("HOST"), "localhost")
func Coalesce[T comparable](values ...T) (v T) {
	var zero T
	for _, v = range values {
		if v != zero {
			return
		}
	}
	return
}
