package e2e

import (
	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Sharing Endpoints", Ordered, func() {
	var shareID string
	var albumID string
	var songID string

	BeforeAll(func() {
		setupTestDB()
		conf.Server.EnableSharing = true

		albums, err := ds.Album(ctx).GetAll(model.QueryOptions{
			Filters: squirrel.Eq{"album.name": "Abbey Road"},
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(albums).ToNot(BeEmpty())
		albumID = albums[0].ID

		songs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
			Filters: squirrel.Eq{"title": "Come Together"},
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(songs).ToNot(BeEmpty())
		songID = songs[0].ID
	})

	It("getShares returns empty initially", func() {
		r := newReq("getShares")
		resp, err := router.GetShares(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.Shares).ToNot(BeNil())
		Expect(resp.Shares.Share).To(BeEmpty())
	})

	It("createShare creates a share for an album", func() {
		r := newReq("createShare", "id", albumID, "description", "Check out this album")
		resp, err := router.CreateShare(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.Shares).ToNot(BeNil())
		Expect(resp.Shares.Share).To(HaveLen(1))

		share := resp.Shares.Share[0]
		Expect(share.ID).ToNot(BeEmpty())
		Expect(share.Description).To(Equal("Check out this album"))
		Expect(share.Username).To(Equal(adminUser.UserName))
		shareID = share.ID
	})

	It("getShares returns the created share", func() {
		r := newReq("getShares")
		resp, err := router.GetShares(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.Shares).ToNot(BeNil())
		Expect(resp.Shares.Share).To(HaveLen(1))

		share := resp.Shares.Share[0]
		Expect(share.ID).To(Equal(shareID))
		Expect(share.Description).To(Equal("Check out this album"))
		Expect(share.Username).To(Equal(adminUser.UserName))
		Expect(share.Entry).ToNot(BeEmpty())
	})

	It("updateShare modifies the description", func() {
		r := newReq("updateShare", "id", shareID, "description", "Updated description")
		resp, err := router.UpdateShare(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Status).To(Equal(responses.StatusOK))

		// Verify update
		r = newReq("getShares")
		resp, err = router.GetShares(r)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Shares.Share).To(HaveLen(1))
		Expect(resp.Shares.Share[0].Description).To(Equal("Updated description"))
	})

	It("deleteShare removes it", func() {
		r := newReq("deleteShare", "id", shareID)
		resp, err := router.DeleteShare(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Status).To(Equal(responses.StatusOK))
	})

	It("getShares returns empty after deletion", func() {
		r := newReq("getShares")
		resp, err := router.GetShares(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.Shares).ToNot(BeNil())
		Expect(resp.Shares.Share).To(BeEmpty())
	})

	It("createShare works with a song ID", func() {
		r := newReq("createShare", "id", songID, "description", "Great song")
		resp, err := router.CreateShare(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.Shares).ToNot(BeNil())
		Expect(resp.Shares.Share).To(HaveLen(1))
		Expect(resp.Shares.Share[0].Description).To(Equal("Great song"))
		Expect(resp.Shares.Share[0].Entry).To(HaveLen(1))
	})

	It("createShare returns error when id parameter is missing", func() {
		r := newReq("createShare")
		_, err := router.CreateShare(r)

		Expect(err).To(HaveOccurred())
	})

	It("updateShare returns error when id parameter is missing", func() {
		r := newReq("updateShare")
		_, err := router.UpdateShare(r)

		Expect(err).To(HaveOccurred())
	})

	It("deleteShare returns error when id parameter is missing", func() {
		r := newReq("deleteShare")
		_, err := router.DeleteShare(r)

		Expect(err).To(HaveOccurred())
	})
})
