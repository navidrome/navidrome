package random

import (
	"errors"
	"slices"
)

// WeightedChooser allows to randomly choose an entry based on their weights
// (higher weight = higher chance of being chosen). Based on the subtraction method described in
// https://eli.thegreenplace.net/2010/01/22/weighted-random-generation-in-python/
type WeightedChooser[T any] struct {
	entries     []T
	weights     []int
	totalWeight int
}

func NewWeightedChooser[T any]() *WeightedChooser[T] {
	return &WeightedChooser[T]{}
}

func (w *WeightedChooser[T]) Add(value T, weight int) {
	w.entries = append(w.entries, value)
	w.weights = append(w.weights, weight)
	w.totalWeight += weight
}

// Pick choose a random entry based on their weights, and removes it from the list
func (w *WeightedChooser[T]) Pick() (T, error) {
	var empty T
	if w.totalWeight == 0 {
		return empty, errors.New("cannot choose from zero weight")
	}
	i, err := w.weightedChoice()
	if err != nil {
		return empty, err
	}
	entry := w.entries[i]
	_ = w.Remove(i)
	return entry, nil
}

func (w *WeightedChooser[T]) weightedChoice() (int, error) {
	if len(w.entries) == 0 {
		return 0, errors.New("cannot choose from empty list")
	}
	rnd := Int64N(w.totalWeight)
	for i, weight := range w.weights {
		rnd -= int64(weight)
		if rnd < 0 {
			return i, nil
		}
	}
	return 0, errors.New("internal error - code should not reach this point")
}

func (w *WeightedChooser[T]) Remove(i int) error {
	if i < 0 || i >= len(w.entries) {
		return errors.New("index out of bounds")
	}

	w.totalWeight -= w.weights[i]

	w.weights = slices.Delete(w.weights, i, i+1)
	w.entries = slices.Delete(w.entries, i, i+1)
	return nil
}

func (w *WeightedChooser[T]) Size() int {
	return len(w.entries)
}
