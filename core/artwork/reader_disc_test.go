package artwork

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Disc Artwork Reader", func() {
	Describe("extractDiscNumber", func() {
		DescribeTable("extracts disc number from filename based on glob pattern",
			func(pattern, filename string, expectedNum int, expectedOk bool) {
				num, ok := extractDiscNumber(pattern, filename)
				Expect(ok).To(Equal(expectedOk))
				if expectedOk {
					Expect(num).To(Equal(expectedNum))
				}
			},
			// Standard disc patterns
			Entry("disc1.jpg", "disc*.*", "disc1.jpg", 1, true),
			Entry("disc2.png", "disc*.*", "disc2.png", 2, true),
			Entry("disc01.jpg", "disc*.*", "disc01.jpg", 1, true),
			Entry("disc02.png", "disc*.*", "disc02.png", 2, true),
			Entry("disc10.jpg", "disc*.*", "disc10.jpg", 10, true),

			// CD patterns
			Entry("cd1.jpg", "cd*.*", "cd1.jpg", 1, true),
			Entry("cd02.png", "cd*.*", "cd02.png", 2, true),

			// No number in filename
			Entry("disc.jpg has no number", "disc*.*", "disc.jpg", 0, false),
			Entry("cd.jpg has no number", "cd*.*", "cd.jpg", 0, false),

			// Extra text after number
			Entry("disc2-bonus.jpg", "disc*.*", "disc2-bonus.jpg", 2, true),
			Entry("disc01_front.png", "disc*.*", "disc01_front.png", 1, true),

			// Case insensitive (filename already lowered by caller)
			Entry("Disc1.jpg lowered", "disc*.*", "disc1.jpg", 1, true),

			// Pattern doesn't match
			Entry("cover.jpg doesn't match disc*.*", "disc*.*", "cover.jpg", 0, false),

			// Pattern with no wildcard before dot
			Entry("front1.jpg with front*.*", "front*.*", "front1.jpg", 1, true),
		)
	})
})
