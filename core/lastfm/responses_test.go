package lastfm

import (
	"encoding/json"
	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LastFM responses", func() {
	Describe("Artist", func() {
		It("parses the response correctly", func() {
			var resp Response
			body, _ := ioutil.ReadFile("tests/fixtures/lastfm.artist.getinfo.json")
			err := json.Unmarshal(body, &resp)
			Expect(err).To(BeNil())

			Expect(resp.Artist.Name).To(Equal("U2"))
			Expect(resp.Artist.MBID).To(Equal("a3cb23fc-acd3-4ce0-8f36-1e5aa6a18432"))
			Expect(resp.Artist.URL).To(Equal("https://www.last.fm/music/U2"))
			Expect(resp.Artist.Bio.Summary).To(ContainSubstring("U2 é uma das mais importantes bandas de rock de todos os tempos"))

			similarArtists := []string{"Passengers", "INXS", "R.E.M.", "Simple Minds", "Bono"}
			for i, similar := range similarArtists {
				Expect(resp.Artist.Similar.Artists[i].Name).To(Equal(similar))
			}
		})
	})

	Describe("SimilarArtists", func() {
		It("parses the response correctly", func() {
			var resp Response
			body, _ := ioutil.ReadFile("tests/fixtures/lastfm.artist.getsimilar.json")
			err := json.Unmarshal(body, &resp)
			Expect(err).To(BeNil())

			Expect(resp.SimilarArtists.Artists).To(HaveLen(2))
			Expect(resp.SimilarArtists.Artists[0].Name).To(Equal("Passengers"))
			Expect(resp.SimilarArtists.Artists[1].Name).To(Equal("INXS"))
		})
	})

	Describe("Error", func() {
		It("parses the error response correctly", func() {
			var error Error
			body := []byte(`{"error":3,"message":"Invalid Method - No method with that name in this package"}`)
			err := json.Unmarshal(body, &error)
			Expect(err).To(BeNil())

			Expect(error.Code).To(Equal(3))
			Expect(error.Message).To(Equal("Invalid Method - No method with that name in this package"))
		})
	})
})
