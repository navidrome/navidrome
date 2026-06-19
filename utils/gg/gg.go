// Package gg implements simple "extensions" to Go language. Based on https://github.com/icza/gog
package gg

// V returns the value of the input pointer, or a zero value if the input pointer is nil.
func V[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

func If[T any](cond bool, v1, v2 T) T {
	if cond {
		return v1
	}
	return v2
}

// Clone returns a pointer to a fresh copy of *p, or nil if p is nil. Use it to
// avoid aliasing the pointed-to value when a separate *T is needed.
func Clone[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
