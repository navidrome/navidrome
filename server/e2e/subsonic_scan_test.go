package e2e

import (
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
		regularUser := createUser("user-2", "regular", "Regular User", false)

		r := newReqWithUser(regularUser, "startScan")
		_, err := router.StartScan(r)

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
