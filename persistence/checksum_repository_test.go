package persistence

import (
	"github.com/cloudsonic/sonic-server/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ChecksumRepository", func() {
	var repo model.ChecksumRepository

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
		sums, err := repo.GetData()
		Expect(err).To(BeNil())
		Expect(sums["b"]).To(Equal("BBB"))
	})

	It("persists data", func() {
		newRepo := NewCheckSumRepository()
		sums, err := newRepo.GetData()
		Expect(err).To(BeNil())
		Expect(sums["b"]).To(Equal("BBB"))
	})
})
