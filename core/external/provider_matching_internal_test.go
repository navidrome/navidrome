package external

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("similarityRatio", func() {
	It("returns 1.0 for identical strings", func() {
		Expect(similarityRatio("hello", "hello")).To(BeNumerically("==", 1.0))
	})

	It("returns 0.0 for empty strings", func() {
		Expect(similarityRatio("", "test")).To(BeNumerically("==", 0.0))
		Expect(similarityRatio("test", "")).To(BeNumerically("==", 0.0))
	})

	It("returns high similarity for remastered suffix", func() {
		// Jaro-Winkler gives ~0.92 for this case
		ratio := similarityRatio("paranoid android", "paranoid android remastered")
		Expect(ratio).To(BeNumerically(">=", 0.85))
	})

	It("returns high similarity for suffix additions like (Live)", func() {
		// Jaro-Winkler gives ~0.96 for this case
		ratio := similarityRatio("bohemian rhapsody", "bohemian rhapsody live")
		Expect(ratio).To(BeNumerically(">=", 0.90))
	})

	It("returns high similarity for 'yesterday' variants (common prefix)", func() {
		// Jaro-Winkler gives ~0.90 because of common prefix
		ratio := similarityRatio("yesterday", "yesterday once more")
		Expect(ratio).To(BeNumerically(">=", 0.85))
	})

	It("returns low similarity for same suffix", func() {
		// Jaro-Winkler gives ~0.70 for this case
		ratio := similarityRatio("postman (live)", "taxman (live)")
		Expect(ratio).To(BeNumerically("<", 0.85))
	})

	It("handles unicode characters", func() {
		ratio := similarityRatio("dont stop believin", "don't stop believin'")
		Expect(ratio).To(BeNumerically(">=", 0.85))
	})

	It("returns low similarity for completely different strings", func() {
		ratio := similarityRatio("abc", "xyz")
		Expect(ratio).To(BeNumerically("<", 0.5))
	})

	It("is symmetric", func() {
		ratio1 := similarityRatio("hello world", "hello")
		ratio2 := similarityRatio("hello", "hello world")
		Expect(ratio1).To(Equal(ratio2))
	})
})
