package e2e

import (
	"net/http"

	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Media Retrieval Endpoints", Ordered, func() {
	BeforeAll(func() {
		setupTestDB()
	})

	Describe("Stream", func() {
		It("returns error when id parameter is missing", func() {
			resp := doReq("stream")

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
		})
	})

	Describe("Download", func() {
		It("returns error when id parameter is missing", func() {
			resp := doReq("download")

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
		})
	})

	Describe("GetCoverArt", func() {
		It("handles request without error", func() {
			w := doRawReq("getCoverArt")

			Expect(w.Code).To(Equal(http.StatusOK))
		})
	})

	Describe("GetAvatar", func() {
		It("returns placeholder avatar when gravatar disabled", func() {
			w := doRawReq("getAvatar", "username", "admin")

			Expect(w.Code).To(Equal(http.StatusOK))
		})
	})

	Describe("GetLyrics", func() {
		It("returns empty lyrics when no match found", func() {
			resp := doReq("getLyrics", "artist", "NonExistentArtist", "title", "NonExistentTitle")

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Lyrics).ToNot(BeNil())
			Expect(resp.Lyrics.Value).To(BeEmpty())
		})
	})

	Describe("GetLyricsBySongId", func() {
		It("returns error when id parameter is missing", func() {
			resp := doReq("getLyricsBySongId")

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
		})

		It("returns error for non-existent song id", func() {
			resp := doReq("getLyricsBySongId", "id", "non-existent-id")

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
		})
	})
})
