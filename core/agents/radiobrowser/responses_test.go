package radiobrowser

import (
	"encoding/json"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func getTime(t string) *time.Time {
	res, _ := time.Parse(time.RFC3339, t)
	return &res
}

var _ = Describe("radio-browser responses", func() {
	Describe("/json/stations", func() {
		var resp RadioStations
		It("parses the response correctly", func() {
			body, _ := os.ReadFile("tests/fixtures/radiobrowser.stations.json")
			err := json.Unmarshal(body, &resp)
			Expect(err).To(BeNil())

			Expect(resp).To(HaveLen(6))
			Expect(resp[0]).To(Equal(RadioStation{
				ChangeId:              "610cafba-71d8-40fc-bf68-1456ec973b9d",
				StationID:             "941ef6f1-0699-4821-95b1-2b678e3ff62e",
				ServerId:              nil,
				Name:                  "\tBest FM",
				Url:                   "http://stream.bestfm.sk/128.mp3",
				UrlResolved:           "http://stream.bestfm.sk/128.mp3",
				Homepage:              "http://bestfm.sk/",
				Favicon:               "",
				Tags:                  "",
				Country:               "Slovakia",
				CountryCode:           "SK",
				IsoCountryCode:        nil,
				State:                 "",
				Language:              "",
				Languagecodes:         "",
				Votes:                 0,
				LastChangeTime:        "2023-01-22 03:34:20",
				IsoLastChangeTime:     getTime("2023-01-22T03:34:20Z"),
				Codec:                 "MP3",
				Bitrate:               128,
				Hls:                   0,
				LastCheckOk:           1,
				LastCheckTime:         "2023-01-22 03:36:33",
				IsoLastCheckTime:      getTime("2023-01-22T03:36:33Z"),
				LastCheckOkTime:       "2023-01-22 03:36:33",
				IsoLastCheckOkTime:    getTime("2023-01-22T03:36:33Z"),
				LastLocalCheckTime:    "",
				IsoLastLocalCheckTime: nil,
				ClickTimestamp:        "2023-01-13 18:13:30",
				IsoClickTimestamp:     getTime("2023-01-13T18:13:30Z"),
				ClickCount:            3,
				ClickTrend:            0,
				SslError:              0,
				GeoLat:                nil,
				GeoLong:               nil,
			}))

			second := resp[1]
			Expect(*second.ServerId).To(Equal("941ef6f1-0699-4821-95b1-2b678e3ff62e"))
			Expect(*second.IsoCountryCode).To(Equal("SK"))
			Expect(*second.GeoLat).To(Equal(95.21))
			Expect(*second.GeoLong).To(Equal(-22.5))
		})
	})

	Describe("/url/stationid", func() {
		var resp UrlResponse
		body, _ := os.ReadFile("tests/fixtures/radiobrowser.url.json")
		err := json.Unmarshal(body, &resp)
		Expect(err).To(BeNil())

		Expect(resp).To(Equal(UrlResponse{
			Ok:        true,
			Message:   "retrieved station url",
			StationId: "6e79967a-aac0-488d-89ec-7656f6db4ca8",
			Name:      "\tNewstalkZBAuckland",
			Url:       "",
		}))
	})
})
