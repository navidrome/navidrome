package metadata

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ffmpegMetadata", func() {
	Context("parseYear", func() {
		It("parses the year correctly", func() {
			var examples = map[string]int{
				"1985":         1985,
				"2002-01":      2002,
				"1969.06":      1969,
				"1980.07.25":   1980,
				"2004-00-00":   2004,
				"2013-May-12":  2013,
				"May 12, 2016": 0,
			}
			for tag, expected := range examples {
				md := &baseMetadata{}
				md.tags = map[string]string{"date": tag}
				Expect(md.Year()).To(Equal(expected))
			}
		})

		It("returns 0 if year is invalid", func() {
			md := &baseMetadata{}
			md.tags = map[string]string{"date": "invalid"}
			Expect(md.Year()).To(Equal(0))
		})
	})
})
