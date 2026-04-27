package e2e

import (
	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Sonic Similarity Endpoints", func() {
	BeforeEach(func() {
		setupTestDB()
	})

	Describe("getSonicSimilarTracks", func() {
		It("returns data not found error when no sonic similarity plugin is available", func() {
			resp := doReq("getSonicSimilarTracks", "id", "any-song-id")

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
			Expect(resp.Error.Code).To(Equal(responses.ErrorDataNotFound))
		})
	})

	Describe("findSonicPath", func() {
		It("returns data not found error when no sonic similarity plugin is available", func() {
			resp := doReq("findSonicPath", "startSongId", "any-song-id", "endSongId", "another-song-id")

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
			Expect(resp.Error.Code).To(Equal(responses.ErrorDataNotFound))
		})
	})
})
