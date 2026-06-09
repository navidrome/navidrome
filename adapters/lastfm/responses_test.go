package lastfm

import (
	"encoding/json"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LastFM responses", func() {
	Describe("Artist", func() {
		It("parses the response correctly", func() {
			var resp Response
			body, _ := os.ReadFile("tests/fixtures/lastfm.artist.getinfo.json")
			err := json.Unmarshal(body, &resp)
			Expect(err).To(BeNil())

			Expect(resp.Artist.Name).To(Equal("U2"))
			Expect(resp.Artist.MBID).To(Equal("a3cb23fc-acd3-4ce0-8f36-1e5aa6a18432"))
			Expect(resp.Artist.URL).To(Equal("https://www.last.fm/music/U2"))
			Expect(resp.Artist.Bio.Summary).To(ContainSubstring("U2 é uma das mais importantes bandas de rock de todos os tempos"))
		})
	})

	Describe("SimilarArtists", func() {
		It("parses the response correctly", func() {
			var resp Response
			body, _ := os.ReadFile("tests/fixtures/lastfm.artist.getsimilar.json")
			err := json.Unmarshal(body, &resp)
			Expect(err).To(BeNil())

			Expect(resp.SimilarArtists.Artists).To(HaveLen(2))
			Expect(resp.SimilarArtists.Artists[0].Name).To(Equal("Passengers"))
			Expect(resp.SimilarArtists.Artists[1].Name).To(Equal("INXS"))
		})
	})

	Describe("TopTracks", func() {
		It("parses the response correctly", func() {
			var resp Response
			body, _ := os.ReadFile("tests/fixtures/lastfm.artist.gettoptracks.json")
			err := json.Unmarshal(body, &resp)
			Expect(err).To(BeNil())

			Expect(resp.TopTracks.Track).To(HaveLen(2))
			Expect(resp.TopTracks.Track[0].Name).To(Equal("Beautiful Day"))
			Expect(resp.TopTracks.Track[0].MBID).To(Equal("f7f264d0-a89b-4682-9cd7-a4e7c37637af"))
			Expect(resp.TopTracks.Track[1].Name).To(Equal("With or Without You"))
			Expect(resp.TopTracks.Track[1].MBID).To(Equal("6b9a509f-6907-4a6e-9345-2f12da09ba4b"))
		})
	})

	Describe("Error", func() {
		It("parses the error response correctly", func() {
			var error Response
			body := []byte(`{"error":3,"message":"Invalid Method - No method with that name in this package"}`)
			err := json.Unmarshal(body, &error)
			Expect(err).To(BeNil())

			Expect(error.Error).To(Equal(3))
			Expect(error.Message).To(Equal("Invalid Method - No method with that name in this package"))
		})
	})
})
