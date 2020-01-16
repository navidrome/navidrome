package persistence

import (
	"github.com/cloudsonic/sonic-server/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MediaFileRepository", func() {
	var repo model.MediaFileRepository

	BeforeEach(func() {
		repo = NewMediaFileRepository()
	})

	Describe("FindByPath", func() {
		It("returns all records from a given ArtistID", func() {
			Expect(repo.FindByPath("/beatles/1")).To(Equal(model.MediaFiles{
				songComeTogether,
			}))
		})
	})

})
