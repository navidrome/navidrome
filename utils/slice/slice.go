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

func ToMap[T any, K comparable, V any](s []T, transformFunc func(T) (K, V)) map[K]V {
	m := map[K]V{}
	for _, item := range s {
		k, v := transformFunc(item)
		m[k] = v
	}
	return m
}

func MostFrequent[T comparable](list []T) T {
	var zero T
	if len(list) == 0 {
		return zero
	}

	counters := make(map[T]int)
	var topItem T
	var topCount int

	for _, value := range list {
		if value == zero {
			continue
		}
		counters[value]++
		if counters[value] > topCount {
			topItem = value
			topCount = counters[value]
		}
	}

	return topItem
}

func Insert[T any](slice []T, value T, index int) []T {
	return append(slice[:index], append([]T{value}, slice[index:]...)...)
}

func Remove[T any](slice []T, index int) []T {
	return append(slice[:index], slice[index+1:]...)
}

func Move[T any](slice []T, srcIndex int, dstIndex int) []T {
	value := slice[srcIndex]
	return Insert(Remove(slice, srcIndex), value, dstIndex)
}

func BreakUp[T any](items []T, chunkSize int) [][]T {
	numTracks := len(items)
	var chunks [][]T
	for i := 0; i < numTracks; i += chunkSize {
		end := i + chunkSize
		if end > numTracks {
			end = numTracks
		}

		chunks = append(chunks, items[i:end])
	}
	return chunks
}

func RangeByChunks[T any](items []T, chunkSize int, cb func([]T) error) error {
	chunks := BreakUp(items, chunkSize)
	for _, chunk := range chunks {
		err := cb(chunk)
		if err != nil {
			return err
		}
	}
	return nil
}
