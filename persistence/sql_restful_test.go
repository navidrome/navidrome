package persistence

import (
	"github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("sqlRestful", func() {
	Describe("parseRestFilters", func() {
		var r sqlRestful
		var options rest.QueryOptions

		BeforeEach(func() {
			r = sqlRestful{}
		})

		It("returns nil if filters is empty", func() {
			options.Filters = nil
			Expect(r.parseRestFilters(options)).To(BeNil())
		})

		It("returns a '=' condition for 'id' filter", func() {
			options.Filters = map[string]interface{}{"id": "123"}
			Expect(r.parseRestFilters(options)).To(Equal(squirrel.And{squirrel.Eq{"id": "123"}}))
		})

		It("returns a 'in' condition for multiples 'id' filters", func() {
			options.Filters = map[string]interface{}{"id": []string{"123", "456"}}
			Expect(r.parseRestFilters(options)).To(Equal(squirrel.And{squirrel.Eq{"id": []string{"123", "456"}}}))
		})

		It("returns a 'like' condition for other filters", func() {
			options.Filters = map[string]interface{}{"name": "joe"}
			Expect(r.parseRestFilters(options)).To(Equal(squirrel.And{squirrel.Like{"name": "joe%"}}))
		})

		It("uses the custom filter", func() {
			r.filterMappings = map[string]filterFunc{
				"test": func(field string, value interface{}) squirrel.Sqlizer {
					return squirrel.Gt{field: value}
				},
			}
			options.Filters = map[string]interface{}{"test": 100}
			Expect(r.parseRestFilters(options)).To(Equal(squirrel.And{squirrel.Gt{"test": 100}}))
		})
	})
})
