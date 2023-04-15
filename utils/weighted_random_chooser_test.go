package utils

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("WeightedRandomChooser", func() {
	var w *WeightedChooser
	BeforeEach(func() {
		w = NewWeightedRandomChooser()
		for i := 0; i < 10; i++ {
			w.Add(i, i)
		}
	})

	It("removes a random item", func() {
		Expect(w.Size()).To(Equal(10))
		_, err := w.GetAndRemove()
		Expect(err).ToNot(HaveOccurred())
		Expect(w.Size()).To(Equal(9))
	})

	It("returns the sole item", func() {
		w = NewWeightedRandomChooser()
		w.Add("a", 1)
		Expect(w.GetAndRemove()).To(Equal("a"))
	})

	It("fails when trying to choose from empty set", func() {
		w = NewWeightedRandomChooser()
		w.Add("a", 1)
		w.Add("b", 1)
		Expect(w.GetAndRemove()).To(BeElementOf("a", "b"))
		Expect(w.GetAndRemove()).To(BeElementOf("a", "b"))
		_, err := w.GetAndRemove()
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
