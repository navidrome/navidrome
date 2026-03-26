package subsonic

import (
	"context"
	"net/http/httptest"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Radio", func() {
	var api *Router
	var ds *tests.MockDataStore
	var ctx context.Context
	var radioRepo *tests.MockedRadioRepo

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		auth.Init(ds)
		api = &Router{ds: ds}
		ctx = context.Background()
		radioRepo = tests.CreateMockedRadioRepo()
		ds.MockedRadio = radioRepo
	})

	Describe("GetInternetRadios", func() {
		BeforeEach(func() {
			radioRepo.All = model.Radios{
				{ID: "rd-1", Name: "Radio 1", StreamUrl: "http://stream1.example.com", HomePageUrl: "http://home1.example.com", UploadedImage: "rd-1_cover.jpg"},
				{ID: "rd-2", Name: "Radio 2", StreamUrl: "http://stream2.example.com"},
			}
		})

		It("returns all radios with basic fields", func() {
			r := httptest.NewRequest("GET", "/rest/getInternetRadios", nil)
			r = r.WithContext(ctx)

			response, err := api.GetInternetRadios(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(response.InternetRadioStations).ToNot(BeNil())
			Expect(response.InternetRadioStations.Radios).To(HaveLen(2))
			Expect(response.InternetRadioStations.Radios[0].ID).To(Equal("rd-1"))
			Expect(response.InternetRadioStations.Radios[0].Name).To(Equal("Radio 1"))
			Expect(response.InternetRadioStations.Radios[0].StreamUrl).To(Equal("http://stream1.example.com"))
			Expect(response.InternetRadioStations.Radios[0].HomepageUrl).To(Equal("http://home1.example.com"))
			Expect(response.InternetRadioStations.Radios[1].ID).To(Equal("rd-2"))
			Expect(response.InternetRadioStations.Radios[1].HomepageUrl).To(BeEmpty())
		})

		Context("with a non-legacy client", func() {
			BeforeEach(func() {
				DeferCleanup(configtest.SetupConfig())
				conf.Server.Subsonic.LegacyClients = "legacy-client"
				player := model.Player{Client: "modern-client"}
				ctx = request.WithPlayer(ctx, player)
			})

			It("includes coverArt from UploadedImage", func() {
				r := httptest.NewRequest("GET", "/rest/getInternetRadios", nil)
				r = r.WithContext(ctx)

				response, err := api.GetInternetRadios(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(response.InternetRadioStations.Radios).To(HaveLen(2))
				Expect(response.InternetRadioStations.Radios[0].OpenSubsonicRadio).ToNot(BeNil())
				Expect(response.InternetRadioStations.Radios[0].CoverArt).To(Equal("rd-1_cover.jpg"))
				Expect(response.InternetRadioStations.Radios[1].OpenSubsonicRadio).ToNot(BeNil())
				Expect(response.InternetRadioStations.Radios[1].CoverArt).To(BeEmpty())
			})
		})

		Context("with a legacy client", func() {
			BeforeEach(func() {
				DeferCleanup(configtest.SetupConfig())
				conf.Server.Subsonic.LegacyClients = "legacy-client"
				player := model.Player{Client: "legacy-client"}
				ctx = request.WithPlayer(ctx, player)
			})

			It("does not include coverArt", func() {
				r := httptest.NewRequest("GET", "/rest/getInternetRadios", nil)
				r = r.WithContext(ctx)

				response, err := api.GetInternetRadios(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(response.InternetRadioStations.Radios).To(HaveLen(2))
				Expect(response.InternetRadioStations.Radios[0].OpenSubsonicRadio).To(BeNil())
				Expect(response.InternetRadioStations.Radios[1].OpenSubsonicRadio).To(BeNil())
			})
		})

		Context("when no player in context", func() {
			It("does not include coverArt (empty client matches legacy list)", func() {
				DeferCleanup(configtest.SetupConfig())
				conf.Server.Subsonic.LegacyClients = "legacy-client"

				r := httptest.NewRequest("GET", "/rest/getInternetRadios", nil)
				r = r.WithContext(ctx)

				response, err := api.GetInternetRadios(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(response.InternetRadioStations.Radios[0].OpenSubsonicRadio).To(BeNil())
			})
		})

		Context("when legacy clients list is empty", func() {
			BeforeEach(func() {
				DeferCleanup(configtest.SetupConfig())
				conf.Server.Subsonic.LegacyClients = ""
				player := model.Player{Client: "any-client"}
				ctx = request.WithPlayer(ctx, player)
			})

			It("includes coverArt for all clients", func() {
				r := httptest.NewRequest("GET", "/rest/getInternetRadios", nil)
				r = r.WithContext(ctx)

				response, err := api.GetInternetRadios(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(response.InternetRadioStations.Radios[0].OpenSubsonicRadio).ToNot(BeNil())
				Expect(response.InternetRadioStations.Radios[0].CoverArt).To(Equal("rd-1_cover.jpg"))
			})
		})

		It("returns error when repository fails", func() {
			radioRepo.SetError(true)

			r := httptest.NewRequest("GET", "/rest/getInternetRadios", nil)
			r = r.WithContext(ctx)

			_, err := api.GetInternetRadios(r)
			Expect(err).To(HaveOccurred())
		})
	})
})
