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
		Expect(resp.Error.Code).To(Equal(responses.ErrorMissingParameter))
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

var _ = Describe("Sharing Downloadable Default", func() {
	var albumID string

	BeforeEach(func() {
		conf.Server.EnableSharing = true
		conf.Server.EnableDownloads = true
		setupTestDB()

		albums, err := ds.Album(ctx).GetAll(model.QueryOptions{
			Filters: squirrel.Eq{"album.name": "Abbey Road"},
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(albums).ToNot(BeEmpty())
		albumID = albums[0].ID
	})

	createShare := func(params ...string) *model.Share {
		resp := doReq("createShare", append([]string{"id", albumID}, params...)...)
		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.Shares.Share).To(HaveLen(1))
		share, err := ds.Share(ctx).Get(resp.Shares.Share[0].ID)
		Expect(err).ToNot(HaveOccurred())
		return share
	}

	It("applies DefaultDownloadableShare when the param is absent", func() {
		conf.Server.DefaultDownloadableShare = true

		Expect(createShare().Downloadable).To(BeTrue())
	})

	It("keeps shares non-downloadable when the default is off", func() {
		conf.Server.DefaultDownloadableShare = false

		Expect(createShare().Downloadable).To(BeFalse())
	})

	It("lets an explicit downloadable=false override the default", func() {
		conf.Server.DefaultDownloadableShare = true

		Expect(createShare("downloadable", "false").Downloadable).To(BeFalse())
	})

	It("lets an explicit downloadable=true override the default", func() {
		conf.Server.DefaultDownloadableShare = false

		Expect(createShare("downloadable", "true").Downloadable).To(BeTrue())
	})

	It("updateShare keeps the current downloadable when the param is absent", func() {
		conf.Server.DefaultDownloadableShare = true
		share := createShare()
		Expect(share.Downloadable).To(BeTrue())

		resp := doReq("updateShare", "id", share.ID, "description", "Updated")
		Expect(resp.Status).To(Equal(responses.StatusOK))

		updated, err := ds.Share(ctx).Get(share.ID)
		Expect(err).ToNot(HaveOccurred())
		Expect(updated.Description).To(Equal("Updated"))
		Expect(updated.Downloadable).To(BeTrue())
	})

	It("updateShare applies an explicit downloadable", func() {
		conf.Server.DefaultDownloadableShare = true
		share := createShare()

		resp := doReq("updateShare", "id", share.ID, "downloadable", "false")
		Expect(resp.Status).To(Equal(responses.StatusOK))

		updated, err := ds.Share(ctx).Get(share.ID)
		Expect(err).ToNot(HaveOccurred())
		Expect(updated.Downloadable).To(BeFalse())
	})
})
