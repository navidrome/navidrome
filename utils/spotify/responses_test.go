package spotify

import (
	"encoding/json"
	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Responses", func() {
	Describe("Search type=artist", func() {
		It("parses the artist search result correctly ", func() {
			var resp SearchResults
			body, _ := ioutil.ReadFile("tests/fixtures/spotify.search.artist.json")
			err := json.Unmarshal(body, &resp)
			Expect(err).To(BeNil())

			Expect(resp.Artists.Items).To(HaveLen(20))
			u2 := resp.Artists.Items[0]
			Expect(u2.Name).To(Equal("U2"))
			Expect(u2.Genres).To(ContainElements("irish rock", "permanent wave", "rock"))
			Expect(u2.ID).To(Equal("51Blml2LZPmy7TTiAg47vQ"))
			Expect(u2.HRef).To(Equal("https://api.spotify.com/v1/artists/51Blml2LZPmy7TTiAg47vQ"))
			Expect(u2.Images[0].URL).To(Equal("https://i.scdn.co/image/e22d5c0c8139b8439440a69854ed66efae91112d"))
			Expect(u2.Images[0].Width).To(Equal(640))
			Expect(u2.Images[0].Height).To(Equal(640))
			Expect(u2.Images[1].URL).To(Equal("https://i.scdn.co/image/40d6c5c14355cfc127b70da221233315497ec91d"))
			Expect(u2.Images[1].Width).To(Equal(320))
			Expect(u2.Images[1].Height).To(Equal(320))
			Expect(u2.Images[2].URL).To(Equal("https://i.scdn.co/image/7293d6752ae8a64e34adee5086858e408185b534"))
			Expect(u2.Images[2].Width).To(Equal(160))
			Expect(u2.Images[2].Height).To(Equal(160))
		})
	})

	Describe("Error", func() {
		It("parses the error response correctly", func() {
			var errorResp Error
			body := []byte(`{"error":"invalid_client","error_description":"Invalid client"}`)
			err := json.Unmarshal(body, &errorResp)
			Expect(err).To(BeNil())

			Expect(errorResp.Code).To(Equal("invalid_client"))
			Expect(errorResp.Message).To(Equal("Invalid client"))
		})
	})
})
