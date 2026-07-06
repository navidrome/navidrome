package jellyfin

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type fakeArtwork struct {
	artwork.Artwork
	recvId string
	data   []byte
}

func (f *fakeArtwork) GetOrPlaceholder(ctx context.Context, id string, size int, square bool) (io.ReadCloser, time.Time, error) {
	f.recvId = id
	data := f.data
	if data == nil {
		data = []byte("IMG")
	}
	return io.NopCloser(bytes.NewReader(data)), time.Now(), nil
}

func newImageRequest(itemId string) (*httptest.ResponseRecorder, *http.Request) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Items/"+itemId+"/Images/Primary", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("itemId", itemId)
	rctx.URLParams.Add("type", "Primary")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	return w, r
}

var _ = Describe("Images", func() {
	It("streams album artwork", func() {
		ds := &tests.MockDataStore{}
		ds.Album(context.Background()).(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "a1", Name: "One"}})
		fa := &fakeArtwork{}
		api := &Router{ds: ds, artwork: fa}

		w, r := newImageRequest(dto.EncodeID("a1"))
		api.getItemImage(w, r)

		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Body.String()).To(Equal("IMG"))
		Expect(fa.recvId).To(ContainSubstring("a1"))
	})

	It("sniffs the Content-Type instead of hardcoding it", func() {
		ds := &tests.MockDataStore{}
		ds.Album(context.Background()).(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "a1", Name: "One"}})

		png := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, make([]byte, 512)...)
		fa := &fakeArtwork{data: png}
		api := &Router{ds: ds, artwork: fa}

		w, r := newImageRequest(dto.EncodeID("a1"))
		api.getItemImage(w, r)

		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Header().Get("Content-Type")).To(Equal("image/png"))
	})
})
