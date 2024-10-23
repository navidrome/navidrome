package slice

import (
	"bufio"
	"bytes"
	"io"
	"iter"
)

func Map[T any, R any](t []T, mapFunc func(T) R) []R {
	r := make([]R, len(t))
	for i, e := range t {
		r[i] = mapFunc(e)
	}
	return r
}

func MapWithArg[I any, O any, A any](t []I, arg A, mapFunc func(A, I) O) []O {
	return Map(t, func(e I) O {
		return mapFunc(arg, e)
	})
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

// LinesFrom returns a Seq that reads lines from the given reader
func LinesFrom(reader io.Reader) iter.Seq[string] {
	return func(yield func(string) bool) {
		scanner := bufio.NewScanner(reader)
		scanner.Split(scanLines)
		for scanner.Scan() {
			if !yield(scanner.Text()) {
				return
			}
		}
	}
}

// From https://stackoverflow.com/a/41433698
func scanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
		if data[i] == '\n' {
			// We have a line terminated by single newline.
			return i + 1, data[0:i], nil
		}
		advance = i + 1
		if len(data) > i+1 && data[i+1] == '\n' {
			advance += 1
		}
		return advance, data[0:i], nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

// CollectChunks collects chunks of n elements from the input sequence and return a Seq of chunks
func CollectChunks[T any](it iter.Seq[T], n int) iter.Seq[[]T] {
	return func(yield func([]T) bool) {
		s := make([]T, 0, n)
		for x := range it {
			s = append(s, x)
			if len(s) >= n {
				if !yield(s) {
					return
				}
				s = make([]T, 0, n)
			}
		}
		if len(s) > 0 {
			yield(s)
		}
	}
}

// SeqFunc returns a Seq that iterates over the slice with the given mapping function
func SeqFunc[I, O any](s []I, f func(I) O) iter.Seq[O] {
	return func(yield func(O) bool) {
		for _, x := range s {
			if !yield(f(x)) {
				return
			}
		}
	}
}
