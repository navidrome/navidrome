package persistence

import (
	"github.com/astaxie/beego/orm"
	. "github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Property Repository", func() {
	var pr model.PropertyRepository

	BeforeEach(func() {
		pr = NewPropertyRepository(NewContext(nil), orm.NewOrm())
	})

	It("saves and restore a new property", func() {
		id := "1"
		value := "a_value"
		Expect(pr.Put(id, value)).To(BeNil())
		Expect(pr.Get(id)).To(Equal("a_value"))
	})

	It("updates a property", func() {
		Expect(pr.Put("1", "another_value")).To(BeNil())
		Expect(pr.Get("1")).To(Equal("another_value"))
	})

	It("returns a default value if property does not exist", func() {
		Expect(pr.DefaultGet("2", "default")).To(Equal("default"))
	})
})
