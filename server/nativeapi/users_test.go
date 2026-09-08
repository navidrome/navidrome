package nativeapi

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func testRouter(ds model.DataStore) http.Handler {
	api := &Router{ds: ds, imgUpload: artwork.NewUploader(ds)}
	r := chi.NewRouter()
	api.addUserRoute(r)
	return r
}

type trackingReader struct {
	r    io.Reader
	read bool
}

func (t *trackingReader) Read(p []byte) (int, error) {
	t.read = true
	return t.r.Read(p)
}

var _ = Describe("User avatar routes", func() {
	var router http.Handler
	var ds *tests.MockDataStore

	newRequest := func(method, path string, body io.Reader, contentType string, asUser model.User) *http.Request {
		r := httptest.NewRequest(method, path, body)
		if contentType != "" {
			r.Header.Set("Content-Type", contentType)
		}
		return r.WithContext(request.WithUser(r.Context(), asUser))
	}

	pngUpload := func() (io.Reader, string) {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		part, err := mw.CreateFormFile("image", "avatar.png")
		Expect(err).ToNot(HaveOccurred())
		Expect(png.Encode(part, image.NewRGBA(image.Rect(0, 0, 8, 8)))).To(Succeed())
		Expect(mw.Close()).To(Succeed())
		return &buf, mw.FormDataContentType()
	}

	regularUser := model.User{ID: "u1", UserName: "regular"}
	adminUser := model.User{ID: "admin", UserName: "admin", IsAdmin: true}
	otherUser := model.User{ID: "u2", UserName: "other"}

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.DataFolder = conf.NewDir(GinkgoT().TempDir())
		conf.Server.EnableUserAvatarUpload = true
		ds = &tests.MockDataStore{}
		Expect(ds.User(context.Background()).Put(&regularUser)).To(Succeed())
		Expect(ds.User(context.Background()).Put(&adminUser)).To(Succeed())
		Expect(ds.User(context.Background()).Put(&otherUser)).To(Succeed())
		router = testRouter(ds)
	})

	It("lets a user upload their own avatar", func() {
		body, ct := pngUpload()
		w := httptest.NewRecorder()
		router.ServeHTTP(w, newRequest("POST", "/user/u1/image", body, ct, regularUser))
		Expect(w.Code).To(Equal(http.StatusOK))

		usr, _ := ds.User(context.Background()).Get("u1")
		Expect(usr.UploadedImage).To(Equal("u1_regular.png"))
	})

	It("lets an admin upload someone else's avatar", func() {
		body, ct := pngUpload()
		w := httptest.NewRecorder()
		router.ServeHTTP(w, newRequest("POST", "/user/u1/image", body, ct, adminUser))
		Expect(w.Code).To(Equal(http.StatusOK))
	})

	It("refuses a third party", func() {
		body, ct := pngUpload()
		w := httptest.NewRecorder()
		router.ServeHTTP(w, newRequest("POST", "/user/u1/image", body, ct, otherUser))
		Expect(w.Code).To(Equal(http.StatusForbidden))
	})

	It("refuses a third party before reading the request body", func() {
		body, ct := pngUpload()
		tracked := &trackingReader{r: body}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, newRequest("POST", "/user/u1/image", tracked, ct, otherUser))
		Expect(w.Code).To(Equal(http.StatusForbidden))
		Expect(tracked.read).To(BeFalse())
	})

	It("refuses everyone, admins included, when the flag is off", func() {
		conf.Server.EnableUserAvatarUpload = false
		body, ct := pngUpload()
		w := httptest.NewRecorder()
		router.ServeHTTP(w, newRequest("POST", "/user/u1/image", body, ct, adminUser))
		Expect(w.Code).To(Equal(http.StatusForbidden))
	})

	It("clears the filename on delete", func() {
		Expect(ds.User(context.Background()).UpdateImage("u1", "u1_regular.png")).To(Succeed())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, newRequest("DELETE", "/user/u1/image", nil, "", regularUser))
		Expect(w.Code).To(Equal(http.StatusOK))

		usr, _ := ds.User(context.Background()).Get("u1")
		Expect(usr.UploadedImage).To(BeEmpty())
	})
})
