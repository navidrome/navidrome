package jellyfin

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type fakeArtwork struct {
	artwork.Artwork
	recvId  string
	recvCtx context.Context
	data    []byte
}

func (f *fakeArtwork) GetOrPlaceholder(ctx context.Context, id string, size int, square bool) (io.ReadCloser, time.Time, error) {
	f.recvId = id
	f.recvCtx = ctx
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

	It("resolves a public playlist id to its cover artwork", func() {
		ds := &tests.MockDataStore{}
		ds.Playlist(context.Background()).(*tests.MockPlaylistRepo).SetData(model.Playlists{{ID: "pl1", Name: "Mix", Public: true}})
		fa := &fakeArtwork{}
		api := &Router{ds: ds, artwork: fa}

		w, r := newImageRequest(dto.EncodeID("pl1"))
		api.getItemImage(w, r)

		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(fa.recvId).To(ContainSubstring("pl1"))
	})

	It("serves the placeholder, not the cover, for a private playlist and an anonymous caller", func() {
		ds := &tests.MockDataStore{}
		ds.Playlist(context.Background()).(*tests.MockPlaylistRepo).SetData(model.Playlists{{ID: "pl1", Name: "Mix", OwnerID: "someone"}})
		fa := &fakeArtwork{}
		api := &Router{ds: ds, artwork: fa}

		w, r := newImageRequest(dto.EncodeID("pl1"))
		api.getItemImage(w, r)

		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(fa.recvId).ToNot(ContainSubstring("pl1"))
	})

	// This endpoint is public (no user in the request), so artwork must be resolved under an
	// elevated context; otherwise a private playlist's cover fails its visibility filter and
	// silently falls back to the placeholder.
	It("resolves artwork under an elevated admin context", func() {
		ds := &tests.MockDataStore{}
		ds.Album(context.Background()).(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "a1", Name: "One"}})
		fa := &fakeArtwork{}
		api := &Router{ds: ds, artwork: fa}

		w, r := newImageRequest(dto.EncodeID("a1"))
		api.getItemImage(w, r)

		Expect(w.Code).To(Equal(http.StatusOK))
		u, ok := request.UserFrom(fa.recvCtx)
		Expect(ok).To(BeTrue())
		Expect(u.IsAdmin).To(BeTrue())
	})
})

// Real image fixtures: postItemImage validates uploads by decoding them.
func pngBytes() []byte {
	var b bytes.Buffer
	Expect(png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 1, 1)))).To(Succeed())
	return b.Bytes()
}

func jpegBytes() []byte {
	var b bytes.Buffer
	Expect(jpeg.Encode(&b, image.NewRGBA(image.Rect(0, 0, 1, 1)), nil)).To(Succeed())
	return b.Bytes()
}

func gifBytes() []byte {
	var b bytes.Buffer
	Expect(gif.Encode(&b, image.NewRGBA(image.Rect(0, 0, 1, 1)), nil)).To(Succeed())
	return b.Bytes()
}

// 1x1 WebP (Go's webp support is decode-only, so this one is pre-encoded).
func webpBytes() []byte {
	b, err := base64.StdEncoding.DecodeString(
		"UklGRjwAAABXRUJQVlA4IDAAAADQAQCdASoBAAEAAgA0JaACdLoB+AADsAD+8Oj3/yC5YXXI1/8gP+QH/ID/+PIAAAA=")
	Expect(err).ToNot(HaveOccurred())
	return b
}

var _ = Describe("postItemImage", func() {
	var api *Router
	var fp *fakePlaylists

	BeforeEach(func() {
		fp = &fakePlaylists{getByIDPls: &model.Playlist{ID: "pl1"}}
		api = &Router{playlists: fp}
	})

	It("uploads a raw JPEG body and returns 204", func() {
		body := jpegBytes()
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/Items/"+dto.EncodeID("pl1")+"/Images/Primary", bytes.NewReader(body))
		r.Header.Set("Content-Type", "image/jpeg")
		r = withChiURLParam(r, "itemId", dto.EncodeID("pl1"))

		api.postItemImage(w, r)

		Expect(w.Code).To(Equal(http.StatusNoContent))
		Expect(fp.setImagePlaylistID).To(Equal("pl1"))
		Expect(fp.setImageBytes).To(Equal(body))
		Expect(fp.setImageExt).To(Equal(".jpeg"))
	})

	It("base64-decodes the body and derives the extension from the actual format, not Content-Type", func() {
		raw := pngBytes()
		encoded := base64.StdEncoding.EncodeToString(raw)
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/Items/"+dto.EncodeID("pl1")+"/Images/Primary", bytes.NewReader([]byte(encoded)))
		r.Header.Set("Content-Type", "image/jpeg") // lies: the payload is a PNG
		r = withChiURLParam(r, "itemId", dto.EncodeID("pl1"))

		api.postItemImage(w, r)

		Expect(w.Code).To(Equal(http.StatusNoContent))
		Expect(fp.setImageBytes).To(Equal(raw))
		Expect(fp.setImageExt).To(Equal(".png"))
	})

	It("returns 501 for a non-playlist item, draining the body first", func() {
		fp.getByIDPls = nil
		fp.getByIDErr = model.ErrNotFound
		bodyReader := bytes.NewReader([]byte("some-bytes-that-must-be-drained"))
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/Items/"+dto.EncodeID("al1")+"/Images/Primary", bodyReader)
		r.Header.Set("Content-Type", "image/jpeg")
		r = withChiURLParam(r, "itemId", dto.EncodeID("al1"))

		api.postItemImage(w, r)

		Expect(w.Code).To(Equal(http.StatusNotImplemented))
		Expect(bodyReader.Len()).To(Equal(0))
	})

	It("returns 500 when the service fails", func() {
		fp.setImageErr = errors.New("boom")
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/Items/"+dto.EncodeID("pl1")+"/Images/Primary", bytes.NewReader(jpegBytes()))
		r.Header.Set("Content-Type", "image/jpeg")
		r = withChiURLParam(r, "itemId", dto.EncodeID("pl1"))

		api.postItemImage(w, r)

		Expect(w.Code).To(Equal(http.StatusInternalServerError))
	})

	It("accepts a raw WebP body", func() {
		body := webpBytes()
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/Items/"+dto.EncodeID("pl1")+"/Images/Primary", bytes.NewReader(body))
		r.Header.Set("Content-Type", "image/webp")
		r = withChiURLParam(r, "itemId", dto.EncodeID("pl1"))

		api.postItemImage(w, r)

		Expect(w.Code).To(Equal(http.StatusNoContent))
		Expect(fp.setImageBytes).To(Equal(body))
		Expect(fp.setImageExt).To(Equal(".webp"))
	})

	It("accepts a raw GIF body", func() {
		body := gifBytes()
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/Items/"+dto.EncodeID("pl1")+"/Images/Primary", bytes.NewReader(body))
		r.Header.Set("Content-Type", "image/gif")
		r = withChiURLParam(r, "itemId", dto.EncodeID("pl1"))

		api.postItemImage(w, r)

		Expect(w.Code).To(Equal(http.StatusNoContent))
		Expect(fp.setImageBytes).To(Equal(body))
		Expect(fp.setImageExt).To(Equal(".gif"))
	})

	It("rejects an oversized body with 400, like the native endpoint", func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.MaxImageUploadSize = "16" // 16 bytes
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/Items/"+dto.EncodeID("pl1")+"/Images/Primary", bytes.NewReader(jpegBytes()))
		r.Header.Set("Content-Type", "image/jpeg")
		r = withChiURLParam(r, "itemId", dto.EncodeID("pl1"))

		api.postItemImage(w, r)

		Expect(w.Code).To(Equal(http.StatusBadRequest))
		Expect(fp.setImagePlaylistID).To(BeEmpty(), "must not persist an over-limit upload")
	})

	It("applies the size limit to the decoded image, not the base64 body", func() {
		DeferCleanup(configtest.SetupConfig())
		img := pngBytes()
		// The raw image is exactly at the limit; its base64 form is 4/3 bigger.
		conf.Server.MaxImageUploadSize = strconv.Itoa(len(img))
		body := []byte(base64.StdEncoding.EncodeToString(img))
		Expect(len(body)).To(BeNumerically(">", len(img)))
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/Items/"+dto.EncodeID("pl1")+"/Images/Primary", bytes.NewReader(body))
		r.Header.Set("Content-Type", "image/png")
		r = withChiURLParam(r, "itemId", dto.EncodeID("pl1"))

		api.postItemImage(w, r)

		Expect(w.Code).To(Equal(http.StatusNoContent))
		Expect(fp.setImageBytes).To(Equal(img))
	})

	It("rejects a base64 body whose decoded image exceeds the limit with 400", func() {
		DeferCleanup(configtest.SetupConfig())
		img := pngBytes()
		conf.Server.MaxImageUploadSize = strconv.Itoa(len(img) - 1)
		body := []byte(base64.StdEncoding.EncodeToString(img))
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/Items/"+dto.EncodeID("pl1")+"/Images/Primary", bytes.NewReader(body))
		r.Header.Set("Content-Type", "image/png")
		r = withChiURLParam(r, "itemId", dto.EncodeID("pl1"))

		api.postItemImage(w, r)

		Expect(w.Code).To(Equal(http.StatusBadRequest))
		Expect(fp.setImagePlaylistID).To(BeEmpty())
	})

	It("rejects a body that is neither an image nor base64 with 400", func() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/Items/"+dto.EncodeID("pl1")+"/Images/Primary", strings.NewReader("!!not base64!!"))
		r.Header.Set("Content-Type", "image/jpeg")
		r = withChiURLParam(r, "itemId", dto.EncodeID("pl1"))

		api.postItemImage(w, r)

		Expect(w.Code).To(Equal(http.StatusBadRequest))
		Expect(fp.setImagePlaylistID).To(BeEmpty())
	})

	It("rejects bytes that sniff as an image but don't decode (e.g. a truncated or renamed file)", func() {
		body := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F'} // JPEG magic, not a JPEG
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/Items/"+dto.EncodeID("pl1")+"/Images/Primary", bytes.NewReader(body))
		r.Header.Set("Content-Type", "image/jpeg")
		r = withChiURLParam(r, "itemId", dto.EncodeID("pl1"))

		api.postItemImage(w, r)

		Expect(w.Code).To(Equal(http.StatusBadRequest))
		Expect(fp.setImagePlaylistID).To(BeEmpty())
	})

	It("forbids a non-admin upload when artwork upload is disabled", func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.EnableArtworkUpload = false
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/Items/"+dto.EncodeID("pl1")+"/Images/Primary", bytes.NewReader(jpegBytes()))
		r.Header.Set("Content-Type", "image/jpeg")
		r = withChiURLParam(r, "itemId", dto.EncodeID("pl1"))
		r = r.WithContext(request.WithUser(r.Context(), model.User{ID: "u1", IsAdmin: false}))

		api.postItemImage(w, r)

		Expect(w.Code).To(Equal(http.StatusForbidden))
		Expect(fp.setImagePlaylistID).To(BeEmpty())
	})

	It("still allows an admin upload when artwork upload is disabled", func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.EnableArtworkUpload = false
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/Items/"+dto.EncodeID("pl1")+"/Images/Primary", bytes.NewReader(jpegBytes()))
		r.Header.Set("Content-Type", "image/jpeg")
		r = withChiURLParam(r, "itemId", dto.EncodeID("pl1"))
		r = r.WithContext(request.WithUser(r.Context(), model.User{ID: "admin", IsAdmin: true}))

		api.postItemImage(w, r)

		Expect(w.Code).To(Equal(http.StatusNoContent))
	})
})

var _ = Describe("deleteItemImage", func() {
	It("removes the playlist image and returns 204", func() {
		fp := &fakePlaylists{getByIDPls: &model.Playlist{ID: "pl1"}}
		api := &Router{playlists: fp}
		w := httptest.NewRecorder()
		r := httptest.NewRequest("DELETE", "/Items/"+dto.EncodeID("pl1")+"/Images/Primary", nil)
		r = withChiURLParam(r, "itemId", dto.EncodeID("pl1"))

		api.deleteItemImage(w, r)

		Expect(w.Code).To(Equal(http.StatusNoContent))
		Expect(fp.removeImagePlaylistID).To(Equal("pl1"))
	})

	It("returns 501 for a non-playlist item", func() {
		fp := &fakePlaylists{getByIDErr: model.ErrNotFound}
		api := &Router{playlists: fp}
		w := httptest.NewRecorder()
		r := httptest.NewRequest("DELETE", "/Items/"+dto.EncodeID("al1")+"/Images/Primary", nil)
		r = withChiURLParam(r, "itemId", dto.EncodeID("al1"))

		api.deleteItemImage(w, r)

		Expect(w.Code).To(Equal(http.StatusNotImplemented))
	})

	It("returns 500 when the service fails", func() {
		fp := &fakePlaylists{getByIDPls: &model.Playlist{ID: "pl1"}, removeImageErr: errors.New("boom")}
		api := &Router{playlists: fp}
		w := httptest.NewRecorder()
		r := httptest.NewRequest("DELETE", "/Items/"+dto.EncodeID("pl1")+"/Images/Primary", nil)
		r = withChiURLParam(r, "itemId", dto.EncodeID("pl1"))

		api.deleteItemImage(w, r)

		Expect(w.Code).To(Equal(http.StatusInternalServerError))
	})
})
