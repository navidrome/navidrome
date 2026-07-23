package public

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/go-chi/jwtauth/v5"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("decodeArtworkID", func() {
	BeforeEach(func() {
		auth.TokenAuth = jwtauth.New("HS256", []byte("super secret"), nil)
	})

	It("fails to decode an invalid token", func() {
		_, err := decodeArtworkID("xx-123")
		Expect(err).To(HaveOccurred())
	})

	It("fails to decode a token without an id", func() {
		token, _ := auth.CreatePublicToken(auth.Claims{})
		_, err := decodeArtworkID(token)
		Expect(err).To(HaveOccurred())
	})
})

type fakeArtwork struct {
	artwork.Service
	img *artwork.Image
	err error
}

func (f *fakeArtwork) Get(context.Context, model.ArtworkID, int, bool) (*artwork.Image, error) {
	return f.img, f.err
}

var _ = Describe("handleImages", func() {
	var w *httptest.ResponseRecorder

	newImageRequest := func(claimID string) *http.Request {
		auth.TokenAuth = jwtauth.New("HS256", []byte("super secret"), nil)
		token, _ := auth.CreatePublicToken(auth.Claims{ID: claimID})
		return httptest.NewRequest("GET", "/img?:id="+url.QueryEscape(token), nil)
	}

	BeforeEach(func() {
		w = httptest.NewRecorder()
	})

	It("returns 404 when the artwork is unavailable", func() {
		pub := &Router{artwork: &fakeArtwork{err: artwork.ErrUnavailable}}
		pub.handleImages(w, newImageRequest("al-1"))
		Expect(w.Code).To(Equal(http.StatusNotFound))
	})

	It("returns 404 when the artwork is not found", func() {
		pub := &Router{artwork: &fakeArtwork{err: model.ErrNotFound}}
		pub.handleImages(w, newImageRequest("al-1"))
		Expect(w.Code).To(Equal(http.StatusNotFound))
	})

	It("serves the image immutable when the token asserts the current hash", func() {
		const hash = "0123456789abcdef"
		img := &artwork.Image{ReadCloser: io.NopCloser(bytes.NewReader([]byte("IMG"))), Hash: hash}
		pub := &Router{artwork: &fakeArtwork{img: img}}
		pub.handleImages(w, newImageRequest("al-1_"+hash))
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Header().Get("Cache-Control")).To(Equal("public, max-age=31536000, immutable"))
		Expect(w.Header().Get("ETag")).To(Equal(`"` + hash + `"`))
	})
})
