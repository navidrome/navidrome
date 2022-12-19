package math2

import "golang.org/x/exp/constraints"

func Min[T constraints.Ordered](vs ...T) T {
	if len(vs) == 0 {
		var zero T
		return zero
	}
	min := vs[0]
	for _, v := range vs[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func Max[T constraints.Ordered](vs ...T) T {
	if len(vs) == 0 {
		var zero T
		return zero
	}
	max := vs[0]
	for _, v := range vs[1:] {
		if v > max {
			max = v
		}
	}
	return max
}
