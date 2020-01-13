package db_sql

import (
	"github.com/cloudsonic/sonic-server/scanner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ChecksumRepository", func() {
	var repo scanner.CheckSumRepository

	BeforeEach(func() {
		Db().Delete(&CheckSums{ID: checkSumId})
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
