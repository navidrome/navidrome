package e2e

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Sessions", func() {
	BeforeEach(func() { setupTestDB() })

	reportBody := func(itemID string) string {
		return `{"ItemId":"` + enc(itemID) + `","PositionTicks":1200000000}`
	}

	Describe("playback reporting", func() {
		It("accepts a playback start report", func() {
			Expect(post("/Sessions/Playing", reportBody(songID("Come Together"))).Code).To(Equal(http.StatusNoContent))
		})

		It("accepts a playback progress report", func() {
			Expect(post("/Sessions/Playing/Progress", reportBody(songID("Come Together"))).Code).To(Equal(http.StatusNoContent))
		})

		It("accepts a playback stopped report and records the play", func() {
			id := songID("So What")
			Expect(post("/Sessions/Playing/Stopped", reportBody(id)).Code).To(Equal(http.StatusNoContent))

			mf, err := ds.MediaFile(ctx).Get(id)
			Expect(err).ToNot(HaveOccurred())
			Expect(mf.PlayCount).To(BeNumerically(">=", 1))
		})
	})

	Describe("capabilities", func() {
		It("acknowledges POST /Sessions/Capabilities", func() {
			Expect(post("/Sessions/Capabilities", "{}").Code).To(Equal(http.StatusNoContent))
		})

		It("acknowledges POST /Sessions/Capabilities/Full", func() {
			Expect(post("/Sessions/Capabilities/Full", "{}").Code).To(Equal(http.StatusNoContent))
		})
	})
})
