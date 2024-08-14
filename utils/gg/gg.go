// Package gg implements simple "extensions" to Go language. Based on https://github.com/icza/gog
package gg

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
