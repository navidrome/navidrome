package e2e

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Sonic Similarity Endpoints", func() {
	BeforeEach(func() {
		setupTestDB()
	})

	Describe("getSonicSimilarTracks", func() {
		It("returns 404 when no sonic similarity plugin is available", func() {
			w := doRawReq("getSonicSimilarTracks", "id", "any-song-id")
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})
	})

	Describe("findSonicPath", func() {
		It("returns 404 when no sonic similarity plugin is available", func() {
			w := doRawReq("findSonicPath", "startSongId", "any-song-id", "endSongId", "another-song-id")
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})
	})
})
