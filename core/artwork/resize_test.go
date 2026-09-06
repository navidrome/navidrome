package artwork

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("resizeStaticImage", func() {
	It("rejects images whose declared dimensions exceed the pixel cap before decoding", func() {
		_, _, err := resizeStaticImage(pngHeaderWithDims(9000, 9000), 300, false)
		Expect(err).To(MatchError(ContainSubstring("exceed pixel cap")))
	})
})
