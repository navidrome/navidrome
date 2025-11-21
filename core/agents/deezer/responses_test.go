package deezer

import (
	"encoding/json"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Responses", func() {
	Describe("Search type=artist", func() {
		It("parses the artist search result correctly ", func() {
			var resp SearchArtistResults
			body, err := os.ReadFile("tests/fixtures/deezer.search.artist.json")
			Expect(err).To(BeNil())
			err = json.Unmarshal(body, &resp)
			Expect(err).To(BeNil())

			Expect(resp.Data).To(HaveLen(17))
			michael := resp.Data[0]
			Expect(michael.Name).To(Equal("Michael Jackson"))
			Expect(michael.PictureXl).To(Equal("https://cdn-images.dzcdn.net/images/artist/97fae13b2b30e4aec2e8c9e0c7839d92/1000x1000-000000-80-0-0.jpg"))
		})
	})

	Describe("Error", func() {
		It("parses the error response correctly", func() {
			var errorResp Error
			body := []byte(`{"error":{"type":"MissingParameterException","message":"Missing parameters: q","code":501}}`)
			err := json.Unmarshal(body, &errorResp)
			Expect(err).To(BeNil())

			Expect(errorResp.Error.Code).To(Equal(501))
			Expect(errorResp.Error.Message).To(Equal("Missing parameters: q"))
		})
	})

	Describe("Related Artists", func() {
		It("parses the related artists response correctly", func() {
			var resp RelatedArtists
			body, err := os.ReadFile("tests/fixtures/deezer.artist.related.json")
			Expect(err).To(BeNil())
			err = json.Unmarshal(body, &resp)
			Expect(err).To(BeNil())

			Expect(resp.Data).To(HaveLen(20))
			justice := resp.Data[0]
			Expect(justice.Name).To(Equal("Justice"))
			Expect(justice.ID).To(Equal(6404))
		})
	})

	Describe("Top Tracks", func() {
		It("parses the top tracks response correctly", func() {
			var resp TopTracks
			body, err := os.ReadFile("tests/fixtures/deezer.artist.top.json")
			Expect(err).To(BeNil())
			err = json.Unmarshal(body, &resp)
			Expect(err).To(BeNil())

			Expect(resp.Data).To(HaveLen(5))
			track := resp.Data[0]
			Expect(track.Title).To(Equal("Instant Crush (feat. Julian Casablancas)"))
			Expect(track.ID).To(Equal(67238732))
			Expect(track.Album.Title).To(Equal("Random Access Memories"))
		})
	})
})
