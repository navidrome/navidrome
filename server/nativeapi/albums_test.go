package nativeapi

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Album Image Endpoints", func() {
	var api *Router
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		api = &Router{ds: &tests.MockDataStore{}, imgUpload: core.NewImageUploadService()}
	})

	DescribeTable("uploadAlbumImage guard",
		func(enableArtworkUpload, isAdmin bool, expectedStatus int) {
			conf.Server.EnableArtworkUpload = enableArtworkUpload
			req := httptest.NewRequest("POST", "/album/al-1/image", nil)
			ctx := request.WithUser(GinkgoT().Context(), model.User{ID: "user-1", IsAdmin: isAdmin})
			w := httptest.NewRecorder()
			api.uploadAlbumImage().ServeHTTP(w, req.WithContext(ctx))
			Expect(w.Code).To(Equal(expectedStatus))
		},
		Entry("enabled, regular user passes guard", true, false, http.StatusBadRequest),
		Entry("enabled, admin passes guard", true, true, http.StatusBadRequest),
		Entry("disabled, admin passes guard", false, true, http.StatusBadRequest),
		Entry("disabled, regular user is forbidden", false, false, http.StatusForbidden),
	)

	DescribeTable("deleteAlbumImage guard",
		func(enableArtworkUpload, isAdmin bool, expectedStatus int) {
			conf.Server.EnableArtworkUpload = enableArtworkUpload
			req := httptest.NewRequest("DELETE", "/album/al-1/image", nil)
			ctx := request.WithUser(GinkgoT().Context(), model.User{ID: "user-1", IsAdmin: isAdmin})
			w := httptest.NewRecorder()
			api.deleteAlbumImage().ServeHTTP(w, req.WithContext(ctx))
			Expect(w.Code).To(Equal(expectedStatus))
		},
		Entry("enabled, regular user passes guard", true, false, http.StatusNotFound),
		Entry("enabled, admin passes guard", true, true, http.StatusNotFound),
		Entry("disabled, admin passes guard", false, true, http.StatusNotFound),
		Entry("disabled, regular user is forbidden", false, false, http.StatusForbidden),
	)
})

// tinyPNG is a valid 1x1 PNG, enough to pass handleImageUpload's image validation.
var tinyPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
	0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x60, 0x64, 0x62, 0x06,
	0x00, 0x00, 0x0e, 0x00, 0x07, 0xd7, 0x6f, 0xe4, 0x78, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e,
	0x44, 0xae, 0x42, 0x60, 0x82,
}

type fakeImgUpload struct {
	entityID, name, oldPath string
}

func (f *fakeImgUpload) SetImage(_ context.Context, _ string, entityID string, name string, oldPath string, _ io.Reader, _ string) (string, error) {
	f.entityID, f.name, f.oldPath = entityID, name, oldPath
	return "stored.png", nil
}

func (f *fakeImgUpload) RemoveImage(_ context.Context, path string) error {
	f.oldPath = path
	return nil
}

var _ = Describe("uploadAlbumImage shared-file handling", func() {
	var api *Router
	var fake *fakeImgUpload

	upload := func(albumID string) {
		body := &bytes.Buffer{}
		w := multipart.NewWriter(body)
		fw, err := w.CreateFormFile("image", "c.png")
		Expect(err).ToNot(HaveOccurred())
		_, _ = fw.Write(tinyPNG)
		Expect(w.Close()).To(Succeed())

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", albumID)
		ctx := request.WithUser(GinkgoT().Context(), model.User{ID: "u", IsAdmin: true})
		ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
		req := httptest.NewRequest("POST", "/album/"+albumID+"/image", body).WithContext(ctx)
		req.Header.Set("Content-Type", w.FormDataContentType())

		rec := httptest.NewRecorder()
		api.uploadAlbumImage().ServeHTTP(rec, req)
		Expect(rec.Code).To(Equal(http.StatusOK))
	}

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		ds := &tests.MockDataStore{}
		ds.Album(GinkgoT().Context()).(*tests.MockAlbumRepo).SetData(model.Albums{
			{ID: "al-1", Name: "Album One", LibraryID: 1, UploadedImage: "shared.jpg"},
			{ID: "al-2", Name: "Album Two", LibraryID: 1, UploadedImage: "shared.jpg"},
			{ID: "al-solo", Name: "Album Solo", LibraryID: 1, UploadedImage: "solo.jpg"},
		})
		fake = &fakeImgUpload{}
		api = &Router{ds: ds, imgUpload: fake}
	})

	It("replaces in place when the file is not shared", func() {
		upload("al-solo")
		Expect(fake.oldPath).ToNot(BeEmpty(), "sole reference: old file should be removed")
		Expect(fake.name).To(Equal("Album Solo"))
	})

	It("keeps the shared file and writes under a unique name", func() {
		upload("al-1")
		Expect(fake.oldPath).To(BeEmpty(), "shared file must not be removed")
		Expect(fake.name).ToNot(Equal("Album One"), "name must be de-duplicated so the derived filename is unique")
		Expect(fake.name).To(HavePrefix("Album One-"))
	})
})
