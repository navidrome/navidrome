package e2e

import (
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Scan Endpoints", func() {
	BeforeEach(func() {
		setupTestDB()
	})

	It("getScanStatus returns status", func() {
		r := newReq("getScanStatus")
		resp, err := router.GetScanStatus(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.ScanStatus).ToNot(BeNil())
		Expect(resp.ScanStatus.Scanning).To(BeFalse())
		Expect(resp.ScanStatus.Count).To(BeNumerically(">", 0))
		Expect(resp.ScanStatus.LastScan).ToNot(BeNil())
	})

	It("startScan requires admin user", func() {
		regularUser := model.User{
			ID:       "user-2",
			UserName: "regular",
			Name:     "Regular User",
			IsAdmin:  false,
		}
		// Store the regular user in the database
		regularUserWithPass := regularUser
		regularUserWithPass.NewPassword = "password"
		Expect(ds.User(ctx).Put(&regularUserWithPass)).To(Succeed())
		Expect(ds.User(ctx).SetUserLibraries(regularUser.ID, []int{lib.ID})).To(Succeed())

		// Reload user with libraries
		loadedUser, err := ds.User(ctx).FindByUsername(regularUser.UserName)
		Expect(err).ToNot(HaveOccurred())
		regularUser.Libraries = loadedUser.Libraries

		r := newReqWithUser(regularUser, "startScan")
		_, err = router.StartScan(r)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not authorized"))
	})

	It("startScan returns scan status response", func() {
		r := newReq("startScan")
		resp, err := router.StartScan(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.ScanStatus).ToNot(BeNil())
	})
})
