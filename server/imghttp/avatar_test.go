package imghttp_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/imghttp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ServeUserAvatar", func() {
	var w *httptest.ResponseRecorder
	var r *http.Request
	var usr model.User

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.DataFolder = conf.NewDir(GinkgoT().TempDir())
		w = httptest.NewRecorder()
		r = httptest.NewRequest("GET", "/avatar", nil)
		usr = model.User{ID: "u1", UserName: "deluan", UpdatedAt: time.Unix(1000, 0)}
	})

	It("returns false when the user has no avatar", func() {
		Expect(imghttp.ServeUserAvatar(w, r, &usr)).To(BeFalse())
	})

	It("returns false when the file is missing on disk", func() {
		usr.UploadedImage = "u1_deluan.png"
		Expect(imghttp.ServeUserAvatar(w, r, &usr)).To(BeFalse())
	})

	It("serves the file with a content type and an ETag", func() {
		usr.UploadedImage = writeAvatar(usr, "png")
		Expect(imghttp.ServeUserAvatar(w, r, &usr)).To(BeTrue())
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Header().Get("Content-Type")).To(Equal("image/png"))
		Expect(w.Header().Get("ETag")).To(Equal(`"` + usr.AvatarTag() + `"`))
	})

	It("answers 304 when the ETag matches", func() {
		usr.UploadedImage = writeAvatar(usr, "png")
		r.Header.Set("If-None-Match", `"`+usr.AvatarTag()+`"`)
		Expect(imghttp.ServeUserAvatar(w, r, &usr)).To(BeTrue())
		Expect(w.Code).To(Equal(http.StatusNotModified))
	})

	It("does not leak the absolute filesystem path in the response", func() {
		usr.UploadedImage = writeAvatar(usr, "png")
		Expect(imghttp.ServeUserAvatar(w, r, &usr)).To(BeTrue())
		Expect(w.Body.String()).NotTo(ContainSubstring(conf.Server.DataFolder.String()))
		for _, values := range w.Header() {
			for _, v := range values {
				Expect(v).NotTo(ContainSubstring(conf.Server.DataFolder.String()))
			}
		}
	})

	It("does not false-304 on an unrelated multi-value If-None-Match", func() {
		usr.UploadedImage = writeAvatar(usr, "png")
		r.Header.Set("If-None-Match", `"deadbeefdeadbeef", "cafecafecafecafe"`)
		Expect(imghttp.ServeUserAvatar(w, r, &usr)).To(BeTrue())
		Expect(w.Code).To(Equal(http.StatusOK))
	})

	It("treats If-None-Match: * as matching the current representation", func() {
		usr.UploadedImage = writeAvatar(usr, "png")
		r.Header.Set("If-None-Match", "*")
		Expect(imghttp.ServeUserAvatar(w, r, &usr)).To(BeTrue())
		Expect(w.Code).To(Equal(http.StatusNotModified))
	})
})

func writeAvatar(u model.User, ext string) string {
	name := u.ID + "_" + u.UserName + "." + ext
	path := filepath.Join(conf.Server.DataFolder.String(), "avatar", name)
	Expect(os.MkdirAll(filepath.Dir(path), 0755)).To(Succeed())
	Expect(os.WriteFile(path, []byte{0x89, 'P', 'N', 'G'}, 0600)).To(Succeed())
	return name
}
