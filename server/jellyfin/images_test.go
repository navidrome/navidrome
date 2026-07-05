package jellyfin

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type fakeArtwork struct {
	artwork.Artwork
	recvId string
}

func (f *fakeArtwork) GetOrPlaceholder(ctx context.Context, id string, size int, square bool) (io.ReadCloser, time.Time, error) {
	f.recvId = id
	return io.NopCloser(strings.NewReader("IMG")), time.Now(), nil
}

var _ = Describe("Images", func() {
	It("streams album artwork", func() {
		ds := &tests.MockDataStore{}
		ds.Album(context.Background()).(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "a1", Name: "One"}})
		fa := &fakeArtwork{}
		api := &Router{ds: ds, artwork: fa}

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/Items/a1/Images/Primary", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("itemId", "a1")
		rctx.URLParams.Add("type", "Primary")
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		api.getItemImage(w, r)

		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Body.String()).To(Equal("IMG"))
		Expect(fa.recvId).To(ContainSubstring("a1"))
	})
})
