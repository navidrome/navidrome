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
		resp := doReq("getInternetRadioStations")

		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.InternetRadioStations).ToNot(BeNil())
		Expect(resp.InternetRadioStations.Radios).To(BeEmpty())
	})

	It("createInternetRadioStation adds a station", func() {
		resp := doReq("createInternetRadioStation",
			"streamUrl", "https://stream.example.com/radio",
			"name", "Test Radio",
			"homepageUrl", "https://example.com",
		)

		Expect(resp.Status).To(Equal(responses.StatusOK))
	})

	It("getInternetRadioStations returns the created station", func() {
		resp := doReq("getInternetRadioStations")

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

	It("getInternetRadioStations remains available to regular users", func() {
		resp := doReqWithUser(regularUser, "getInternetRadioStations")

		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.InternetRadioStations).ToNot(BeNil())
		Expect(resp.InternetRadioStations.Radios).To(HaveLen(1))
		Expect(resp.InternetRadioStations.Radios[0].Name).To(Equal("Test Radio"))
	})

	It("createInternetRadioStation requires admin user", func() {
		resp := doReqWithUser(regularUser, "createInternetRadioStation",
			"streamUrl", "https://stream.example.com/hacked",
			"name", "Hacked Radio",
		)

		Expect(resp.Status).To(Equal(responses.StatusFailed))
		Expect(resp.Error).ToNot(BeNil())
		Expect(resp.Error.Code).To(Equal(responses.ErrorAuthorizationFail))

		resp = doReq("getInternetRadioStations")
		Expect(resp.InternetRadioStations.Radios).To(HaveLen(1))
		Expect(resp.InternetRadioStations.Radios[0].Name).To(Equal("Test Radio"))
	})

	It("updateInternetRadioStation modifies the station", func() {
		resp := doReq("updateInternetRadioStation",
			"id", radioID,
			"streamUrl", "https://stream.example.com/radio-v2",
			"name", "Updated Radio",
			"homepageUrl", "https://updated.example.com",
		)

		Expect(resp.Status).To(Equal(responses.StatusOK))

		// Verify update
		resp = doReq("getInternetRadioStations")
		Expect(resp.InternetRadioStations.Radios).To(HaveLen(1))
		Expect(resp.InternetRadioStations.Radios[0].Name).To(Equal("Updated Radio"))
		Expect(resp.InternetRadioStations.Radios[0].StreamUrl).To(Equal("https://stream.example.com/radio-v2"))
		Expect(resp.InternetRadioStations.Radios[0].HomepageUrl).To(Equal("https://updated.example.com"))
	})

	It("updateInternetRadioStation requires admin user", func() {
		resp := doReqWithUser(regularUser, "updateInternetRadioStation",
			"id", radioID,
			"streamUrl", "https://stream.example.com/hacked",
			"name", "Hacked Radio",
		)

		Expect(resp.Status).To(Equal(responses.StatusFailed))
		Expect(resp.Error).ToNot(BeNil())
		Expect(resp.Error.Code).To(Equal(responses.ErrorAuthorizationFail))

		resp = doReq("getInternetRadioStations")
		Expect(resp.InternetRadioStations.Radios).To(HaveLen(1))
		Expect(resp.InternetRadioStations.Radios[0].Name).To(Equal("Updated Radio"))
		Expect(resp.InternetRadioStations.Radios[0].StreamUrl).To(Equal("https://stream.example.com/radio-v2"))
	})

	It("deleteInternetRadioStation requires admin user", func() {
		resp := doReqWithUser(regularUser, "deleteInternetRadioStation", "id", radioID)

		Expect(resp.Status).To(Equal(responses.StatusFailed))
		Expect(resp.Error).ToNot(BeNil())
		Expect(resp.Error.Code).To(Equal(responses.ErrorAuthorizationFail))

		resp = doReq("getInternetRadioStations")
		Expect(resp.InternetRadioStations.Radios).To(HaveLen(1))
		Expect(resp.InternetRadioStations.Radios[0].ID).To(Equal(radioID))
	})

	It("deleteInternetRadioStation removes it", func() {
		resp := doReq("deleteInternetRadioStation", "id", radioID)

		Expect(resp.Status).To(Equal(responses.StatusOK))
	})

	It("getInternetRadioStations returns empty after deletion", func() {
		resp := doReq("getInternetRadioStations")

		Expect(resp.Status).To(Equal(responses.StatusOK))
		Expect(resp.InternetRadioStations).ToNot(BeNil())
		Expect(resp.InternetRadioStations.Radios).To(BeEmpty())
	})
})
