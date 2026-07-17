package e2e

import (
	"net/http"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Sessions", func() {
	BeforeEach(func() { setupTestDB() })

	ticks := func(ms int64) int64 { return ms * 10_000 }
	reportBody := func(itemID string, positionTicks int64) string {
		return `{"ItemId":"` + enc(itemID) + `","PositionTicks":` + strconv.FormatInt(positionTicks, 10) + `}`
	}

	Describe("playback reporting", func() {
		It("accepts a playback start report", func() {
			Expect(post("/Sessions/Playing", reportBody(songID("Come Together"), 0)).Code).To(Equal(http.StatusNoContent))
		})

		It("accepts a playback progress report", func() {
			Expect(post("/Sessions/Playing/Progress", reportBody(songID("Come Together"), ticks(5000))).Code).To(Equal(http.StatusNoContent))
		})

		It("counts a play stopped past the threshold", func() {
			id := songID("So What")
			mf, err := ds.MediaFile(ctx).Get(id)
			Expect(err).ToNot(HaveOccurred())
			// Report a stop at the end of the track — comfortably past 50% / the 4-minute cap.
			Expect(post("/Sessions/Playing/Stopped", reportBody(id, ticks(int64(mf.Duration*1000)))).Code).To(Equal(http.StatusNoContent))

			mf, err = ds.MediaFile(ctx).Get(id)
			Expect(err).ToNot(HaveOccurred())
			Expect(mf.PlayCount).To(BeNumerically(">=", 1))
		})

		It("does not count a brief play stopped before the threshold", func() {
			// Regression: Finamp sends a Stopped report on every track switch, so an immediate skip
			// (1 second in) must not mark the track played. Seeded tracks are >= 120s, so the 50%
			// threshold is always well above 1s.
			id := songID("Help!")
			Expect(post("/Sessions/Playing/Stopped", reportBody(id, ticks(1000))).Code).To(Equal(http.StatusNoContent))

			mf, err := ds.MediaFile(ctx).Get(id)
			Expect(err).ToNot(HaveOccurred())
			Expect(mf.PlayCount).To(Equal(int64(0)))
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
