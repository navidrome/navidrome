package random

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("WeightedChooser", func() {
	var w *WeightedChooser[int]
	BeforeEach(func() {
		w = NewWeightedChooser[int]()
		for i := 0; i < 10; i++ {
			w.Add(i, i+1)
		}
	})

	It("selects and removes a random item", func() {
		Expect(w.Size()).To(Equal(10))
		_, err := w.Pick()
		Expect(err).ToNot(HaveOccurred())
		Expect(w.Size()).To(Equal(9))
	})

	It("removes items", func() {
		Expect(w.Size()).To(Equal(10))
		for i := 0; i < 10; i++ {
			Expect(w.Remove(0)).To(Succeed())
		}
		Expect(w.Size()).To(Equal(0))
	})

	It("returns error if trying to remove an invalid index", func() {
		Expect(w.Size()).To(Equal(10))
		Expect(w.Remove(-1)).ToNot(Succeed())
		Expect(w.Remove(10000)).ToNot(Succeed())
		Expect(w.Size()).To(Equal(10))
	})

	It("returns the sole item", func() {
		ws := NewWeightedChooser[string]()
		ws.Add("a", 1)
		Expect(ws.Pick()).To(Equal("a"))
	})

	It("returns all items from the list", func() {
		for i := 0; i < 10; i++ {
			Expect(w.Pick()).To(BeElementOf(0, 1, 2, 3, 4, 5, 6, 7, 8, 9))
		}
		Expect(w.Size()).To(Equal(0))
	})

	It("fails when trying to choose from empty set", func() {
		w = NewWeightedChooser[int]()
		w.Add(1, 1)
		w.Add(2, 1)
		Expect(w.Pick()).To(BeElementOf(1, 2))
		Expect(w.Pick()).To(BeElementOf(1, 2))
		_, err := w.Pick()
		Expect(err).To(HaveOccurred())
	})

	It("chooses based on weights", func() {
		counts := [10]int{}
		for i := 0; i < 200000; i++ {
			c, _ := w.weightedChoice()
			counts[c] = counts[c] + 1
		}
		for i := 0; i < 9; i++ {
			Expect(counts[i]).To(BeNumerically("<", counts[i+1]))
		}
	})
})
