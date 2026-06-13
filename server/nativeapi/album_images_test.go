package nativeapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type fakeArtwork struct {
	images []artwork.AlbumImageInfo
	err    error
}

func (f *fakeArtwork) Get(context.Context, model.ArtworkID, int, bool) (io.ReadCloser, time.Time, error) {
	return nil, time.Time{}, nil
}

func (f *fakeArtwork) GetOrPlaceholder(context.Context, string, int, bool) (io.ReadCloser, time.Time, error) {
	return nil, time.Time{}, nil
}

func (f *fakeArtwork) AlbumImages(context.Context, string) ([]artwork.AlbumImageInfo, error) {
	return f.images, f.err
}

var _ = Describe("albumImages handler", func() {
	var (
		router http.Handler
		aw     *fakeArtwork
	)

	BeforeEach(func() {
		aw = &fakeArtwork{}
		api := &Router{artwork: aw}
		r := chi.NewRouter()
		api.addAlbumImagesRoute(r)
		router = r
	})

	doGet := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		return w
	}

	It("returns the list of album images as JSON", func() {
		aw.images = []artwork.AlbumImageInfo{
			{CoverArt: "al-abc_0", Type: "Front"},
			{CoverArt: "al-abc:1_0", Type: "Back", Name: "back.jpg"},
		}

		w := doGet("/album/abc/images")

		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Header().Get("Content-Type")).To(ContainSubstring("application/json"))
		Expect(w.Body.String()).To(ContainSubstring(`"coverArt":"al-abc:1_0"`))
		Expect(w.Body.String()).To(ContainSubstring(`"type":"Back"`))
		Expect(w.Body.String()).To(ContainSubstring(`"name":"back.jpg"`))
	})

	It("returns 404 when the album is not found", func() {
		aw.err = model.ErrNotFound

		w := doGet("/album/missing/images")

		Expect(w.Code).To(Equal(http.StatusNotFound))
	})
})
