package metadata

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("legacyReleaseDate", func() {

	DescribeTable("legacyReleaseDate",
		func(recordingDate, originalDate, releaseDate, expected string) {
			md := New("", Info{
				Tags: map[string][]string{
					"DATE":         {recordingDate},
					"ORIGINALDATE": {originalDate},
					"RELEASEDATE":  {releaseDate},
				},
			})

			result := legacyReleaseDate(md)
			Expect(result).To(Equal(expected))
		},
		Entry("regular mapping", "2020-05-15", "2019-02-10", "2021-01-01", "2021-01-01"),
		Entry("legacy mapping", "2020-05-15", "2019-02-10", "", "2020-05-15"),
		Entry("legacy mapping, originalYear < year", "2018-05-15", "2019-02-10", "2021-01-01", "2021-01-01"),
		Entry("legacy mapping, originalYear empty", "2020-05-15", "", "2021-01-01", "2021-01-01"),
		Entry("legacy mapping, releaseYear", "2020-05-15", "2019-02-10", "2021-01-01", "2021-01-01"),
		Entry("legacy mapping, same dates", "2020-05-15", "2020-05-15", "", "2020-05-15"),
	)
})
