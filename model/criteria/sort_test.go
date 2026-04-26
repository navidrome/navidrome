package criteria

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = Describe("OrderByFields", func() {
	It("defaults to title ascending when Sort is empty", func() {
		c := Criteria{}
		gomega.Expect(c.OrderByFields()).To(gomega.Equal([]SortField{{Field: "title", Desc: false}}))
	})

	It("parses a single field", func() {
		c := Criteria{Sort: "title"}
		gomega.Expect(c.OrderByFields()).To(gomega.Equal([]SortField{{Field: "title", Desc: false}}))
	})

	It("parses descending prefix", func() {
		c := Criteria{Sort: "-rating"}
		gomega.Expect(c.OrderByFields()).To(gomega.Equal([]SortField{{Field: "rating", Desc: true}}))
	})

	It("parses ascending prefix", func() {
		c := Criteria{Sort: "+title"}
		gomega.Expect(c.OrderByFields()).To(gomega.Equal([]SortField{{Field: "title", Desc: false}}))
	})

	It("parses multiple comma-separated fields", func() {
		c := Criteria{Sort: "title,-rating"}
		gomega.Expect(c.OrderByFields()).To(gomega.Equal([]SortField{
			{Field: "title", Desc: false},
			{Field: "rating", Desc: true},
		}))
	})

	It("inverts directions when Order is desc", func() {
		c := Criteria{Sort: "-date,title", Order: "desc"}
		gomega.Expect(c.OrderByFields()).To(gomega.Equal([]SortField{
			{Field: "date", Desc: false},
			{Field: "title", Desc: true},
		}))
	})

	It("skips invalid fields", func() {
		c := Criteria{Sort: "bogus,title"}
		gomega.Expect(c.OrderByFields()).To(gomega.Equal([]SortField{{Field: "title", Desc: false}}))
	})

	It("falls back to title when all fields are invalid", func() {
		c := Criteria{Sort: "bogus,invalid"}
		gomega.Expect(c.OrderByFields()).To(gomega.Equal([]SortField{{Field: "title", Desc: false}}))
	})

	It("resolves tag aliases (albumtype -> releasetype)", func() {
		c := Criteria{Sort: "albumtype"}
		gomega.Expect(c.OrderByFields()).To(gomega.Equal([]SortField{{Field: "releasetype", Desc: false}}))
	})

	It("resolves field aliases (recordingdate -> date)", func() {
		AddTagNames([]string{"recordingdate"})
		c := Criteria{Sort: "recordingdate"}
		gomega.Expect(c.OrderByFields()).To(gomega.Equal([]SortField{{Field: "date", Desc: false}}))
	})

	It("handles the random field", func() {
		c := Criteria{Sort: "random"}
		gomega.Expect(c.OrderByFields()).To(gomega.Equal([]SortField{{Field: "random", Desc: false}}))
	})

	It("ignores invalid Order value", func() {
		c := Criteria{Sort: "-title", Order: "invalid"}
		gomega.Expect(c.OrderByFields()).To(gomega.Equal([]SortField{{Field: "title", Desc: true}}))
	})

	It("handles whitespace in fields", func() {
		c := Criteria{Sort: " title , -rating "}
		gomega.Expect(c.OrderByFields()).To(gomega.Equal([]SortField{
			{Field: "title", Desc: false},
			{Field: "rating", Desc: true},
		}))
	})

	It("skips empty parts from trailing commas", func() {
		c := Criteria{Sort: "title,,rating,"}
		gomega.Expect(c.OrderByFields()).To(gomega.Equal([]SortField{
			{Field: "title", Desc: false},
			{Field: "rating", Desc: false},
		}))
	})
})

var _ = Describe("SortFieldNames", func() {
	It("returns canonical field names", func() {
		c := Criteria{Sort: "title,-rating,albumtype"}
		gomega.Expect(c.SortFieldNames()).To(gomega.Equal([]string{"title", "rating", "releasetype"}))
	})

	It("defaults to title when Sort is empty", func() {
		c := Criteria{}
		gomega.Expect(c.SortFieldNames()).To(gomega.Equal([]string{"title"}))
	})
})
