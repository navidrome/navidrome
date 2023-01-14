package utils

import (
	"errors"

	"github.com/navidrome/navidrome/utils/number"
)

type WeightedChooser struct {
	entries     []interface{}
	weights     []int
	totalWeight int
}

func NewWeightedRandomChooser() *WeightedChooser {
	return &WeightedChooser{}
}

func (w *WeightedChooser) Add(value interface{}, weight int) {
	w.entries = append(w.entries, value)
	w.weights = append(w.weights, weight)
	w.totalWeight += weight
}

// GetAndRemove choose a random entry based on their weights, and removes it from the list
func (w *WeightedChooser) GetAndRemove() (interface{}, error) {
	if w.totalWeight == 0 {
		return nil, errors.New("cannot choose from zero weight")
	}
	i, err := w.weightedChoice()
	if err != nil {
		return nil, err
	}
	entry := w.entries[i]
	w.Remove(i)
	return entry, nil
}

// Based on https://eli.thegreenplace.net/2010/01/22/weighted-random-generation-in-python/
func (w *WeightedChooser) weightedChoice() (int, error) {
	if w.totalWeight == 0 {
		return 0, errors.New("no choices available")
	}
	rnd := number.RandomInt64(int64(w.totalWeight))
	for i, weight := range w.weights {
		rnd -= int64(weight)
		if rnd < 0 {
			return i, nil
		}
	}
	return 0, errors.New("internal error - code should not reach this point")
}

func (w *WeightedChooser) Remove(i int) {
	w.totalWeight -= w.weights[i]

	w.weights[i] = w.weights[len(w.weights)-1]
	w.weights = w.weights[:len(w.weights)-1]

	w.entries[i] = w.entries[len(w.entries)-1]
	w.entries[len(w.entries)-1] = nil
	w.entries = w.entries[:len(w.entries)-1]
}

func (w *WeightedChooser) Size() int {
	return len(w.entries)
}
