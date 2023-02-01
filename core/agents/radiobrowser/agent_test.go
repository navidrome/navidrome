package radiobrowser

import (
	"context"
	"net/http"
	"os"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RadioBrowserAgent", func() {
	var ds model.DataStore
	var ctx context.Context

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		ctx = context.Background()
	})

	Describe("RadioBrowserAgentCostructor", func() {
		It("takes base URL from config", func() {
			conf.Server.RadioBrowser.BaseUrl = "https://example.com:8080"
			agent := RadioBrowserConstructor(ds)
			Expect(agent.baseUrl).To(Equal("https://example.com:8080"))
		})
	})

	Describe("GetRadioInfo", func() {
		var agent *radioBrowserAgent
		var httpClient *tests.FakeHttpClient

		BeforeEach(func() {
			agent = RadioBrowserConstructor(ds)
			httpClient = &tests.FakeHttpClient{}
			client := NewClient("BASE_URL", httpClient)
			agent.client = client
		})

		It("Updates the database", func() {
			f, _ := os.Open("tests/fixtures/radiobrowser.stations.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}
			Expect(agent.GetRadioInfo(ctx)).To(BeNil())
			count, err := ds.RadioInfo(ctx).CountAll()
			Expect(err).To(BeNil())
			Expect(count).To(Equal(int64(5)))
			data, err := ds.RadioInfo(ctx).Get("6e79967a-aac0-488d-89ec-7656f6db4ca8")
			Expect(err).To(BeNil())
			Expect(data.Url).To(Equal("https://ais-nzme.streamguys1.com/nz_002_aac"))

			Expect(ds.Property(ctx).Get(model.PropLastRefresh)).NotTo(BeNil())
		})

		It("Purges existing entries", func() {
			f, _ := os.Open("tests/fixtures/radiobrowser.stations.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}
			_ = ds.RadioInfo(ctx).Insert(&model.RadioInfo{
				ID:       "123",
				Name:     "Example Radio",
				Url:      "https://example.com/stream",
				Homepage: "https://example.com",
				Favicon:  "https://example.com/favicon.ico",
				Existing: false,
				BaseRadioInfo: model.BaseRadioInfo{
					Tags:        "tag1,tag2,tag3",
					Country:     "nowhere",
					CountryCode: "XX",
					Codec:       "MP3",
					Bitrate:     192,
				},
			})

			Expect(agent.GetRadioInfo(ctx)).To(BeNil())
			count, err := ds.RadioInfo(ctx).CountAll()
			Expect(err).To(BeNil())
			Expect(count).To(Equal(int64(5)))
			data, err := ds.RadioInfo(ctx).Get("123")
			Expect(data).To(BeNil())
			Expect(err).ToNot(BeNil())
		})
	})
})
