package e2e

import (
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
			w, r := newRawReq("stream")
			_, err := router.Stream(w, r)

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Download", func() {
		It("returns error when id parameter is missing", func() {
			w, r := newRawReq("download")
			_, err := router.Download(w, r)

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("GetCoverArt", func() {
		It("handles request without error", func() {
			w, r := newRawReq("getCoverArt")
			_, err := router.GetCoverArt(w, r)

			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("GetAvatar", func() {
		It("returns placeholder avatar when gravatar disabled", func() {
			w, r := newRawReq("getAvatar", "username", "admin")
			resp, err := router.GetAvatar(w, r)

			// When gravatar is disabled, it returns nil response (writes directly to w)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).To(BeNil())
		})
	})

	Describe("GetLyrics", func() {
		It("returns empty lyrics when no match found", func() {
			r := newReq("getLyrics", "artist", "NonExistentArtist", "title", "NonExistentTitle")
			resp, err := router.GetLyrics(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Lyrics).ToNot(BeNil())
			Expect(resp.Lyrics.Value).To(BeEmpty())
		})
	})

	Describe("GetLyricsBySongId", func() {
		It("returns error when id parameter is missing", func() {
			r := newReq("getLyricsBySongId")
			_, err := router.GetLyricsBySongId(r)

			Expect(err).To(HaveOccurred())
		})

		It("returns error for non-existent song id", func() {
			r := newReq("getLyricsBySongId", "id", "non-existent-id")
			_, err := router.GetLyricsBySongId(r)

			Expect(err).To(HaveOccurred())
		})
	})
})
