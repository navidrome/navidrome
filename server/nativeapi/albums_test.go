package nativeapi

import (
	"net/http"
	"net/http/httptest"

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
