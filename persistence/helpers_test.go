package persistence

import (
	"github.com/Masterminds/squirrel"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Helpers", func() {
	Describe("Exists", func() {
		It("constructs the correct EXISTS query", func() {
			e := exists("album", squirrel.Eq{"id": 1})
			sql, args, err := e.ToSql()
			Expect(sql).To(Equal("exists (select 1 from album where id = ?)"))
			Expect(args).To(Equal([]interface{}{1}))
			Expect(err).To(BeNil())
		})
	})
})
