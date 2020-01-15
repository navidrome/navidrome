package persistence

import (
	"github.com/cloudsonic/sonic-server/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ChecksumRepository", func() {
	var repo model.CheckSumRepository

	BeforeEach(func() {
		Db().Delete(&Checksum{ID: checkSumId})
		repo = NewCheckSumRepository()
		err := repo.SetData(map[string]string{
			"a": "AAA", "b": "BBB",
		})
		if err != nil {
			panic(err)
		}
	})

	It("can retrieve data", func() {
		Expect(repo.Get("b")).To(Equal("BBB"))
	})

	It("persists data", func() {
		newRepo := NewCheckSumRepository()
		Expect(newRepo.Get("b")).To(Equal("BBB"))
	})
})
