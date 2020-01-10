package storm

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type Empty struct {
	ID        string
	Something int
}

type User struct {
	ID   string
	Name string `db:"index"`
}

var _ = Describe("Domain Tagging", func() {

	It("does not change a struct that does not have any tag", func() {
		empty := &Empty{}
		tagged := tag(empty)
		Expect(getStructTag(tagged, "ID")).To(BeEmpty())
		Expect(getStructTag(tagged, "Something")).To(BeEmpty())
	})

	It("adds index to indexed fields", func() {
		user := &User{}
		tagged := tag(user)
		Expect(getStructTag(tagged, "ID")).To(BeEmpty())
		Expect(getStructTag(tagged, "Name")).To(Equal(`storm:"index"`))
	})
})
