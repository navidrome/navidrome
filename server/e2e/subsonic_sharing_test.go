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
		conf.Server.EnableSharing = true
		setupTestDB()

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
		resp := doReq("getShares")

		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.Shares).ToNot(BeNil())
		Expect(resp.Shares.Share).To(BeEmpty())
	})

	It("createShare creates a share for an album", func() {
		resp := doReq("createShare", "id", albumID, "description", "Check out this album")

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
		resp := doReq("getShares")

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
		resp := doReq("updateShare", "id", shareID, "description", "Updated description")

		Expect(resp.Status).To(Equal(responses.StatusOK))

		// Verify update
		resp = doReq("getShares")
		Expect(resp.Shares.Share).To(HaveLen(1))
		Expect(resp.Shares.Share[0].Description).To(Equal("Updated description"))
	})

	It("deleteShare removes it", func() {
		resp := doReq("deleteShare", "id", shareID)

		Expect(resp.Status).To(Equal(responses.StatusOK))
	})

	It("getShares returns empty after deletion", func() {
		resp := doReq("getShares")

		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.Shares).ToNot(BeNil())
		Expect(resp.Shares.Share).To(BeEmpty())
	})

	It("createShare works with a song ID", func() {
		resp := doReq("createShare", "id", songID, "description", "Great song")

		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.Shares).ToNot(BeNil())
		Expect(resp.Shares.Share).To(HaveLen(1))
		Expect(resp.Shares.Share[0].Description).To(Equal("Great song"))
		Expect(resp.Shares.Share[0].Entry).To(HaveLen(1))
	})

	It("createShare returns error when id parameter is missing", func() {
		resp := doReq("createShare")

		Expect(resp.Status).To(Equal(responses.StatusFailed))
		Expect(resp.Error).ToNot(BeNil())
	})

	It("updateShare returns error when id parameter is missing", func() {
		resp := doReq("updateShare")

		Expect(resp.Status).To(Equal(responses.StatusFailed))
		Expect(resp.Error).ToNot(BeNil())
	})

	It("deleteShare returns error when id parameter is missing", func() {
		resp := doReq("deleteShare")

		Expect(resp.Status).To(Equal(responses.StatusFailed))
		Expect(resp.Error).ToNot(BeNil())
	})
})

var _ = Describe("Sharing Cross-User Isolation", Ordered, func() {
	var userA, userB model.User
	var shareID string
	var albumID string

	BeforeAll(func() {
		conf.Server.EnableSharing = true
		setupTestDB()

		userA = createUser("share-user-a", "share-user-a", "Share User A", false)
		userB = createUser("share-user-b", "share-user-b", "Share User B", false)

		albums, err := ds.Album(ctx).GetAll(model.QueryOptions{
			Filters: squirrel.Eq{"album.name": "Abbey Road"},
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(albums).ToNot(BeEmpty())
		albumID = albums[0].ID

		resp := doReqWithUser(userA, "createShare", "id", albumID, "description", "User A's share")
		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.Shares.Share).To(HaveLen(1))
		shareID = resp.Shares.Share[0].ID
		Expect(resp.Shares.Share[0].Username).To(Equal(userA.UserName))
	})

	It("userB's getShares does not leak userA's share", func() {
		resp := doReqWithUser(userB, "getShares")

		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.Shares).ToNot(BeNil())
		Expect(resp.Shares.Share).To(BeEmpty())
	})

	It("userA still sees own share", func() {
		resp := doReqWithUser(userA, "getShares")

		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.Shares.Share).To(HaveLen(1))
		Expect(resp.Shares.Share[0].ID).To(Equal(shareID))
		Expect(resp.Shares.Share[0].Description).To(Equal("User A's share"))
	})

	It("admin sees userA's share", func() {
		resp := doReqWithUser(adminUser, "getShares")

		Expect(resp.Status).To(Equal(responses.StatusOK))
		ids := make([]string, len(resp.Shares.Share))
		for i, s := range resp.Shares.Share {
			ids[i] = s.ID
		}
		Expect(ids).To(ContainElement(shareID))
	})

	It("userB cannot updateShare on userA's share", func() {
		resp := doReqWithUser(userB, "updateShare", "id", shareID, "description", "hijacked")

		Expect(resp.Status).To(Equal(responses.StatusFailed))
		Expect(resp.Error).ToNot(BeNil())

		// Confirm description unchanged for userA.
		check := doReqWithUser(userA, "getShares")
		Expect(check.Shares.Share).To(HaveLen(1))
		Expect(check.Shares.Share[0].Description).To(Equal("User A's share"))
	})

	It("userB cannot deleteShare on userA's share", func() {
		resp := doReqWithUser(userB, "deleteShare", "id", shareID)

		Expect(resp.Status).To(Equal(responses.StatusFailed))
		Expect(resp.Error).ToNot(BeNil())

		// Confirm share still present for userA.
		check := doReqWithUser(userA, "getShares")
		Expect(check.Shares.Share).To(HaveLen(1))
		Expect(check.Shares.Share[0].ID).To(Equal(shareID))
	})
})
