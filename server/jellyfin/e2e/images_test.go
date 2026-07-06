package e2e

import (
	"net/http"

	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// The image endpoint is public and resolves artwork under an elevated (admin) context. The suite
// wires a spyArtwork that captures the resolved ArtworkID and the context, so these tests assert
// resolution and elevation without needing real image processing.
var _ = Describe("Item images", func() {
	BeforeEach(func() { setupTestDB() })

	It("resolves an album's Primary image", func() {
		id := albumID("Abbey Road")
		w := get("/Items/" + enc(id) + "/Images/Primary")
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Body.String()).To(Equal("IMG"))
		Expect(artworkSpy.lastID).To(ContainSubstring(id))
	})

	It("resolves an artist's Primary image", func() {
		id := artistID("Miles Davis")
		Expect(get("/Items/" + enc(id) + "/Images/Primary").Code).To(Equal(http.StatusOK))
		Expect(artworkSpy.lastID).To(ContainSubstring(id))
	})

	It("resolves a private playlist's cover for its owner under an elevated context", func() {
		// The route carries no user in ctx (public); the owner is identified by the request token,
		// and resolution then runs elevated so the visibility filter doesn't eat the cover.
		plID := createPlaylist("Private Mix", nil)
		w := get("/Items/" + enc(plID) + "/Images/Primary")
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(artworkSpy.lastID).To(ContainSubstring(plID))

		u, ok := request.UserFrom(artworkSpy.lastCtx)
		Expect(ok).To(BeTrue())
		Expect(u.IsAdmin).To(BeTrue())
	})

	It("serves images without authentication (public route)", func() {
		id := albumID("IV")
		w := rawReq("GET", "/Items/"+enc(id)+"/Images/Primary", "")
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Body.String()).To(Equal("IMG"))
	})

	Describe("private playlist covers", func() {
		It("does not resolve a private playlist's cover for an unauthenticated caller", func() {
			plID := createPlaylist("Secret Mix", nil) // owned by admin, private
			w := rawReq("GET", "/Items/"+enc(plID)+"/Images/Primary", "")
			Expect(w.Code).To(Equal(http.StatusOK)) // placeholder, not an auth error
			Expect(artworkSpy.lastID).ToNot(ContainSubstring(plID))
		})

		It("does not resolve a private playlist's cover for another user", func() {
			plID := createPlaylist("Secret Mix", nil)
			w := getAs(regularUser, "/Items/"+enc(plID)+"/Images/Primary")
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(artworkSpy.lastID).ToNot(ContainSubstring(plID))
		})

		It("resolves a public playlist's cover for anyone", func() {
			plID := createPlaylist("Shared Mix", nil)
			Expect(post("/Playlists/"+enc(plID), `{"IsPublic":true}`).Code).To(Equal(http.StatusNoContent))
			w := rawReq("GET", "/Items/"+enc(plID)+"/Images/Primary", "")
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(artworkSpy.lastID).To(ContainSubstring(plID))
		})
	})
})
