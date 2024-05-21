package persistence

import (
	"context"

	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Property Repository", func() {
	var pr model.PropertyRepository

	BeforeEach(func() {
		pr = NewPropertyRepository(log.NewContext(context.TODO()), NewDBXBuilder(db.Db()))
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
