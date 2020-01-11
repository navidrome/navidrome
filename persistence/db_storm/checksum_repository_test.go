package db_storm

import (
	"github.com/cloudsonic/sonic-server/scanner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ChecksumRepository", func() {
	var repo scanner.CheckSumRepository

	BeforeEach(func() {
		Db().Drop(checkSumBucket)
		repo = NewCheckSumRepository()
		repo.SetData(map[string]string{
			"a": "AAA", "b": "BBB",
		})
	})

	It("can retrieve data", func() {
		Expect(repo.Get("b")).To(Equal("BBB"))
	})

	It("persists data", func() {
		newRepo := NewCheckSumRepository()
		Expect(newRepo.Get("b")).To(Equal("BBB"))
	})
})
