package slice

func Map[T any, R any](t []T, mapFunc func(T) R) []R {
	r := make([]R, len(t))
	for i, e := range t {
		r[i] = mapFunc(e)
	}
	return r
}

func Group[T any, K comparable](s []T, keyFunc func(T) K) map[K][]T {
	m := map[K][]T{}
	for _, item := range s {
		k := keyFunc(item)
		m[k] = append(m[k], item)
	}
	return m
}

func MostFrequent[T comparable](list []T) T {
	if len(list) == 0 {
		var zero T
		return zero
	}
	var topItem T
	var topCount int
	counters := map[T]int{}

	if len(list) == 1 {
		topItem = list[0]
	} else {
		for _, id := range list {
			c := counters[id] + 1
			counters[id] = c
			if c > topCount {
				topItem = id
				topCount = c
			}
		}
	}

	return topItem
}
