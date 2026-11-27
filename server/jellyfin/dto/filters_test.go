package dto

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("QueryFiltersLegacy", func() {
	It("marshals all four keys, empty ones as [] not null", func() {
		b, err := json.Marshal(QueryFiltersLegacy{
			Genres: []string{"Rock"}, Tags: []string{}, OfficialRatings: []string{}, Years: []int{1999},
		})
		Expect(err).ToNot(HaveOccurred())
		j := string(b)
		Expect(j).To(ContainSubstring(`"Genres":["Rock"]`))
		Expect(j).To(ContainSubstring(`"Tags":[]`))
		Expect(j).To(ContainSubstring(`"OfficialRatings":[]`))
		Expect(j).To(ContainSubstring(`"Years":[1999]`))
	})
})
