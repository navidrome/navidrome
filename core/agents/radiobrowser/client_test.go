package radiobrowser

import (
	"context"
	"net/http"
	"os"

	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {
	var httpClient *tests.FakeHttpClient
	var client *Client
	BeforeEach(func() {
		httpClient = &tests.FakeHttpClient{}
		client = NewClient("BASE_URL", httpClient)
	})

	Describe("GetAllRadios", func() {
		It("parses a response correctly", func() {
			f, _ := os.Open("tests/fixtures/radiobrowser.stations.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}

			radios, err := client.GetAllRadios(context.Background())

			Expect(err).To(BeNil())
			Expect(*radios).To(HaveLen(6))
			Expect((*radios)[5].StationID).To(Equal("98152a6d-9a3d-46d7-8019-c8c03f29be38"))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal("BASE_URL/json/stations?hidebroken=true"))
		})
	})
})
