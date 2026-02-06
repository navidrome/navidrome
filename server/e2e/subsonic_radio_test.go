package e2e

import (
	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Internet Radio Endpoints", Ordered, func() {
	var radioID string

	BeforeAll(func() {
		setupTestDB()
	})

	It("getInternetRadioStations returns empty initially", func() {
		r := newReq("getInternetRadioStations")
		resp, err := router.GetInternetRadios(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.InternetRadioStations).ToNot(BeNil())
		Expect(resp.InternetRadioStations.Radios).To(BeEmpty())
	})

	It("createInternetRadioStation adds a station", func() {
		r := newReq("createInternetRadioStation",
			"streamUrl", "https://stream.example.com/radio",
			"name", "Test Radio",
			"homepageUrl", "https://example.com",
		)
		resp, err := router.CreateInternetRadio(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Status).To(Equal(responses.StatusOK))
	})

	It("getInternetRadioStations returns the created station", func() {
		r := newReq("getInternetRadioStations")
		resp, err := router.GetInternetRadios(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.InternetRadioStations).ToNot(BeNil())
		Expect(resp.InternetRadioStations.Radios).To(HaveLen(1))

		radio := resp.InternetRadioStations.Radios[0]
		Expect(radio.Name).To(Equal("Test Radio"))
		Expect(radio.StreamUrl).To(Equal("https://stream.example.com/radio"))
		Expect(radio.HomepageUrl).To(Equal("https://example.com"))
		radioID = radio.ID
		Expect(radioID).ToNot(BeEmpty())
	})

	It("updateInternetRadioStation modifies the station", func() {
		r := newReq("updateInternetRadioStation",
			"id", radioID,
			"streamUrl", "https://stream.example.com/radio-v2",
			"name", "Updated Radio",
			"homepageUrl", "https://updated.example.com",
		)
		resp, err := router.UpdateInternetRadio(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Status).To(Equal(responses.StatusOK))

		// Verify update
		r = newReq("getInternetRadioStations")
		resp, err = router.GetInternetRadios(r)
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.InternetRadioStations.Radios).To(HaveLen(1))
		Expect(resp.InternetRadioStations.Radios[0].Name).To(Equal("Updated Radio"))
		Expect(resp.InternetRadioStations.Radios[0].StreamUrl).To(Equal("https://stream.example.com/radio-v2"))
		Expect(resp.InternetRadioStations.Radios[0].HomepageUrl).To(Equal("https://updated.example.com"))
	})

	It("deleteInternetRadioStation removes it", func() {
		r := newReq("deleteInternetRadioStation", "id", radioID)
		resp, err := router.DeleteInternetRadio(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Status).To(Equal(responses.StatusOK))
	})

	It("getInternetRadioStations returns empty after deletion", func() {
		r := newReq("getInternetRadioStations")
		resp, err := router.GetInternetRadios(r)

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.InternetRadioStations).ToNot(BeNil())
		Expect(resp.InternetRadioStations.Radios).To(BeEmpty())
	})
})
