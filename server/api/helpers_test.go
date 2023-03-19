package api

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BuildPaginationLinksAndMeta", func() {
	var (
		totalItems   int32
		params       GetTracksParams
		resourceName string
	)

	BeforeEach(func() {
		totalItems = 150
		resourceName = "api/resource"
	})

	Context("with default page limit and offset", func() {
		BeforeEach(func() {
			l, o := int32(10), int32(0)
			params = GetTracksParams{
				PageLimit:  &l,
				PageOffset: &o,
			}
		})

		It("returns correct pagination links and meta", func() {
			links, meta := buildPaginationLinksAndMeta(totalItems, params, resourceName)

			Expect(links.First).To(Equal(p("api/resource?page[offset]=0&page[limit]=10")))
			Expect(links.Last).To(Equal(p("api/resource?page[offset]=140&page[limit]=10")))
			Expect(links.Next).To(Equal(p("api/resource?page[offset]=10&page[limit]=10")))
			Expect(links.Prev).To(BeNil())

			Expect(meta.CurrentPage).To(Equal(p(int32(1))))
			Expect(meta.TotalItems).To(Equal(p(int32(150))))
			Expect(meta.TotalPages).To(Equal(p(int32(15))))
		})
	})

	Context("with custom page limit and offset", func() {
		BeforeEach(func() {
			params = GetTracksParams{
				PageLimit:  p((PageLimit)(20)),
				PageOffset: p((PageOffset)(40)),
			}
		})

		It("returns correct pagination links and meta", func() {
			links, meta := buildPaginationLinksAndMeta(totalItems, params, resourceName)

			Expect(links.First).To(Equal(p("api/resource?page[offset]=0&page[limit]=20")))
			Expect(links.Last).To(Equal(p("api/resource?page[offset]=140&page[limit]=20")))
			Expect(links.Next).To(Equal(p("api/resource?page[offset]=60&page[limit]=20")))
			Expect(links.Prev).To(Equal(p("api/resource?page[offset]=20&page[limit]=20")))

			Expect(meta.CurrentPage).To(Equal(p(int32(3))))
			Expect(meta.TotalItems).To(Equal(p(int32(150))))
			Expect(meta.TotalPages).To(Equal(p(int32(8))))
		})
	})

	Context("with various filter params", func() {
		BeforeEach(func() {
			params = GetTracksParams{
				PageLimit:            p((PageLimit)(20)),
				PageOffset:           p((PageOffset)(40)),
				FilterEquals:         &[]string{"property1:value1", "property2:value2"},
				FilterContains:       &[]string{"property3:value3"},
				FilterLessThan:       &[]string{"property4:value4"},
				FilterLessOrEqual:    &[]string{"property5:value5"},
				FilterGreaterThan:    &[]string{"property6:value6"},
				FilterGreaterOrEqual: &[]string{"property7:value7"},
				FilterStartsWith:     &[]string{"property8:value8"},
				FilterEndsWith:       &[]string{"property9:value9"},
			}
		})

		It("returns correct pagination links with filter params", func() {
			links, _ := buildPaginationLinksAndMeta(totalItems, params, resourceName)

			expectedLinkPrefix := "api/resource?"
			expectedParams := []string{
				"page[offset]=0&page[limit]=20",
				"filter[equals]=property1:value1&filter[equals]=property2:value2",
				"filter[contains]=property3:value3",
				"filter[lessThan]=property4:value4",
				"filter[lessOrEqual]=property5:value5",
				"filter[greaterThan]=property6:value6",
				"filter[greaterOrEqual]=property7:value7",
				"filter[startsWith]=property8:value8",
				"filter[endsWith]=property9:value9",
			}

			Expect(*links.First).To(HavePrefix(expectedLinkPrefix))
			Expect(*links.Last).To(HavePrefix(expectedLinkPrefix))
			Expect(*links.Next).To(HavePrefix(expectedLinkPrefix))
			Expect(*links.Prev).To(HavePrefix(expectedLinkPrefix))

			for _, param := range expectedParams {
				Expect(*links.First).To(ContainSubstring(param))
				Expect(*links.Last).To(ContainSubstring(param))
				Expect(*links.Next).To(ContainSubstring(param))
				Expect(*links.Prev).To(ContainSubstring(param))
			}
		})
	})
})
