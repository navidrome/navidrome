package persistence

import (
	"github.com/cloudsonic/sonic-server/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PropertyRepository", func() {
	var repo model.PropertyRepository

	BeforeEach(func() {
		repo = NewPropertyRepository()
		repo.(*propertyRepository).DeleteAll()
	})

	It("saves and retrieves data", func() {
		Expect(repo.Put("1", "test")).To(BeNil())
		Expect(repo.Get("1")).To(Equal("test"))
	})

	It("returns default if data is not found", func() {
		Expect(repo.DefaultGet("2", "default")).To(Equal("default"))
	})

	It("returns value if found", func() {
		Expect(repo.Put("3", "test")).To(BeNil())
		Expect(repo.DefaultGet("3", "default")).To(Equal("test"))
	})
})
