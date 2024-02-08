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

// FirstOr is a generic helper function that returns the first non-zero value from
// a list of comparable values, or a default value if all the values are zero.
func FirstOr[T comparable](or T, values ...T) T {
	// Initialize a zero value of the same type as the input values.
	var zero T

	// Loop through each input value and check if it is non-zero. If a non-zero value
	// is found, return it immediately.
	for _, v := range values {
		if v != zero {
			return v
		}
	}

	// If all the input values are zero, return the default value.
	return or
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
