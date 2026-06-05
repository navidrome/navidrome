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
		resp := doReq("getScanStatus")

		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.ScanStatus).ToNot(BeNil())
		Expect(resp.ScanStatus.Scanning).To(BeFalse())
		Expect(resp.ScanStatus.Count).To(BeNumerically(">", 0))
		Expect(resp.ScanStatus.LastScan).ToNot(BeNil())
	})

	It("startScan requires admin user", func() {
		resp := doReqWithUser(regularUser, "startScan")

		Expect(resp.Status).To(Equal(responses.StatusFailed))
		Expect(resp.Error).ToNot(BeNil())
	})

	It("startScan returns scan status response", func() {
		resp := doReq("startScan")

		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.ScanStatus).ToNot(BeNil())
	})
})
