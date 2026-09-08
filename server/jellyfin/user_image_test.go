package jellyfin

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// writeUserAvatar seeds a fake avatar file on disk and returns the UploadedImage filename to store
// on the user record.
func writeUserAvatar(u *model.User) string {
	name := u.ID + "_" + u.UserName + ".png"
	path := filepath.Join(conf.Server.DataFolder.String(), consts.AvatarFolder, name)
	Expect(os.MkdirAll(filepath.Dir(path), 0755)).To(Succeed())
	Expect(os.WriteFile(path, []byte{0x89, 'P', 'N', 'G'}, 0600)).To(Succeed())
	return name
}

var _ = Describe("GET /userimage", func() {
	var api *Router
	var ds *tests.MockDataStore
	var ur *tests.MockedUserRepo
	var pub, priv, noAvatar *model.User

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.DataFolder = conf.NewDir(GinkgoT().TempDir())
		conf.Server.Jellyfin.ExposedPublicUsers = "publicuser"

		ds = &tests.MockDataStore{}
		auth.Init(ds)
		ur = ds.User(context.Background()).(*tests.MockedUserRepo)

		pub = &model.User{ID: testID("pub1"), UserName: "publicuser"}
		pub.UploadedImage = writeUserAvatar(pub)
		Expect(ur.Put(pub)).To(Succeed())

		// Has a real avatar so the anonymous-401 test below fails on a served image (200), not a
		// 404, if the isPublicUser gate is ever removed.
		priv = &model.User{ID: testID("u1"), UserName: "alice"}
		priv.UploadedImage = writeUserAvatar(priv)
		Expect(ur.Put(priv)).To(Succeed())

		noAvatar = &model.User{ID: testID("u4"), UserName: "carol"}
		Expect(ur.Put(noAvatar)).To(Succeed())

		api = New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	})

	get := func(query string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		api.ServeHTTP(w, httptest.NewRequest("GET", "/userimage"+query, nil))
		return w
	}

	It("serves an allowlisted user's avatar without authentication", func() {
		w := get("?userId=" + dto.EncodeID(pub.ID))
		Expect(w.Code).To(Equal(http.StatusOK))
	})

	// This is the leak this whole design exists to prevent: anonymous access must be denied for
	// anyone not on the allowlist, not just fall through to a 404.
	It("refuses an anonymous request for a user not on the allowlist with 401", func() {
		w := get("?userId=" + dto.EncodeID(priv.ID))
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
	})

	It("refuses every anonymous request when the allowlist is empty", func() {
		conf.Server.Jellyfin.ExposedPublicUsers = ""
		w := get("?userId=" + dto.EncodeID(pub.ID))
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
	})

	It("rejects an anonymous request with no userId with 400", func() {
		w := get("")
		Expect(w.Code).To(Equal(http.StatusBadRequest))
	})

	It("returns 404 for an authenticated user with no avatar", func() {
		tok, err := auth.CreateToken(noAvatar)
		Expect(err).ToNot(HaveOccurred())
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/userimage?userId="+dto.EncodeID(noAvatar.ID), nil)
		r.Header.Set("X-Emby-Token", tok)
		api.ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusNotFound))
	})

	It("matches an allowlist entry regardless of surrounding whitespace and casing", func() {
		conf.Server.Jellyfin.ExposedPublicUsers = "  PublicUser  "
		w := get("?userId=" + dto.EncodeID(pub.ID))
		Expect(w.Code).To(Equal(http.StatusOK))
	})

	It("404s an authenticated request for an id that decodes but does not exist", func() {
		tok, err := auth.CreateToken(priv)
		Expect(err).ToNot(HaveOccurred())
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/userimage?userId="+dto.EncodeID(testID("ghost")), nil)
		r.Header.Set("X-Emby-Token", tok)
		api.ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusNotFound))
	})
})

var _ = Describe("POST /userimage and DELETE /userimage", func() {
	var router *Router
	var ds *tests.MockDataStore
	var ur *tests.MockedUserRepo
	var caller *model.User

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.DataFolder = conf.NewDir(GinkgoT().TempDir())
		conf.Server.EnableUserAvatarUpload = true

		ds = &tests.MockDataStore{}
		auth.Init(ds)
		ur = ds.User(context.Background()).(*tests.MockedUserRepo)

		caller = &model.User{ID: "u1", UserName: "regular"}
		Expect(ur.Put(caller)).To(Succeed())
		// A real canonical id, unlike caller's literal "u1", so it round-trips through dto.EncodeID.
		Expect(ur.Put(&model.User{ID: testID("u2"), UserName: "other"})).To(Succeed())

		router = New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, artwork.NewUploader(ds))
	})

	tokenFor := func(u *model.User) string {
		tok, err := auth.CreateToken(u)
		Expect(err).ToNot(HaveOccurred())
		return tok
	}
	authenticatedRequestWithBody := func(method, target string, body io.Reader) *http.Request {
		r := httptest.NewRequest(method, target, body)
		r.Header.Set("X-Emby-Token", tokenFor(caller))
		return r
	}
	authenticatedRequest := func(method, target string) *http.Request {
		return authenticatedRequestWithBody(method, target, nil)
	}
	pngBytes := func() []byte {
		var buf bytes.Buffer
		Expect(png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 8, 8)))).To(Succeed())
		return buf.Bytes()
	}

	Describe("POST /userimage", func() {
		It("accepts a base64 body with a charset suffix", func() {
			body := base64.StdEncoding.EncodeToString(pngBytes())
			r := authenticatedRequestWithBody("POST", "/userimage", strings.NewReader(body))
			r.Header.Set("Content-Type", "image/png; charset=utf-8")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusNoContent))

			usr, _ := ds.User(context.Background()).Get("u1")
			Expect(usr.UploadedImage).To(Equal("u1_regular.png"))
		})

		It("accepts raw image bytes too", func() {
			r := authenticatedRequestWithBody("POST", "/userimage", bytes.NewReader(pngBytes()))
			r.Header.Set("Content-Type", "image/png")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusNoContent))
		})

		It("rejects raw bytes over MaxImageUploadSize instead of letting SetAvatar truncate them", func() {
			img := pngBytes()
			// One byte under the image size: still well inside the base64-inflation read cap
			// (limit*4/3+4), so only the post-decode size check can catch this.
			conf.Server.MaxImageUploadSize = strconv.Itoa(len(img)-1) + "B"
			r := authenticatedRequestWithBody("POST", "/userimage", bytes.NewReader(img))
			r.Header.Set("Content-Type", "image/png")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusBadRequest))

			usr, _ := ds.User(context.Background()).Get("u1")
			Expect(usr.UploadedImage).To(BeEmpty())
		})

		It("rejects a body that is not a decodable image with 400", func() {
			body := base64.StdEncoding.EncodeToString([]byte("this is not an image"))
			r := authenticatedRequestWithBody("POST", "/userimage", strings.NewReader(body))
			r.Header.Set("Content-Type", "image/png")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusBadRequest))

			usr, _ := ds.User(context.Background()).Get("u1")
			Expect(usr.UploadedImage).To(BeEmpty())
		})

		It("rejects an unknown content type", func() {
			r := authenticatedRequestWithBody("POST", "/userimage", strings.NewReader("x"))
			r.Header.Set("Content-Type", "text/plain")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("refuses a third party", func() {
			body := base64.StdEncoding.EncodeToString(pngBytes())
			r := authenticatedRequestWithBody("POST", "/userimage?userId="+dto.EncodeID(testID("u2")), strings.NewReader(body))
			r.Header.Set("Content-Type", "image/png")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusForbidden))
		})

		It("refuses even an admin when the flag is off", func() {
			conf.Server.EnableUserAvatarUpload = false
			admin := &model.User{ID: "admin1", UserName: "boss", IsAdmin: true}
			Expect(ur.Put(admin)).To(Succeed())

			body := base64.StdEncoding.EncodeToString(pngBytes())
			r := httptest.NewRequest("POST", "/userimage", strings.NewReader(body))
			r.Header.Set("X-Emby-Token", tokenFor(admin))
			r.Header.Set("Content-Type", "image/png")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusForbidden))
		})

		It("is not reachable anonymously", func() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest("POST", "/userimage", strings.NewReader("x")))
			Expect(w.Code).To(Equal(http.StatusUnauthorized))
		})
	})

	Describe("DELETE /userimage", func() {
		It("clears the avatar and removes the file from disk", func() {
			name := writeUserAvatar(caller)
			Expect(ur.UpdateImage(caller.ID, name)).To(Succeed())
			path := caller.UploadedImagePath()

			w := httptest.NewRecorder()
			router.ServeHTTP(w, authenticatedRequest("DELETE", "/userimage"))
			Expect(w.Code).To(Equal(http.StatusNoContent))

			usr, _ := ds.User(context.Background()).Get("u1")
			Expect(usr.UploadedImage).To(BeEmpty())
			_, err := os.Stat(path)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})
})

var _ = Describe("isPublicUser", func() {
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
	})

	It("matches a configured name case-insensitively and trims whitespace", func() {
		conf.Server.Jellyfin.ExposedPublicUsers = " Alice ,bob"
		Expect(isPublicUser("alice")).To(BeTrue())
		Expect(isPublicUser("ALICE")).To(BeTrue())
		Expect(isPublicUser("bob")).To(BeTrue())
	})

	It("rejects a user not on the allowlist", func() {
		conf.Server.Jellyfin.ExposedPublicUsers = "alice"
		Expect(isPublicUser("eve")).To(BeFalse())
	})

	It("rejects everyone when the allowlist is empty", func() {
		conf.Server.Jellyfin.ExposedPublicUsers = ""
		Expect(isPublicUser("alice")).To(BeFalse())
	})
})

var _ = Describe("PrimaryImageTag", func() {
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
	})

	It("is present on userToDto when the user has an avatar", func() {
		u := model.User{ID: "u1", UserName: "deluan", UploadedImage: "u1_deluan.png"}
		got := userToDto(&u, "srv", "sid")
		Expect(got.PrimaryImageTag).To(Equal(u.AvatarTag()))
		Expect(got.PrimaryImageTag).ToNot(BeEmpty())
	})

	It("is empty on userToDto when the user has no avatar", func() {
		u := model.User{ID: "u1", UserName: "deluan"}
		Expect(userToDto(&u, "srv", "sid").PrimaryImageTag).To(BeEmpty())
	})

	It("is present on getPublicUsers when the listed user has an avatar", func() {
		ds := &tests.MockDataStore{}
		ur := ds.User(context.Background()).(*tests.MockedUserRepo)
		u := &model.User{ID: testID("pub1"), UserName: "publicuser", UploadedImage: "x.png"}
		Expect(ur.Put(u)).To(Succeed())
		conf.Server.Jellyfin.ExposedPublicUsers = "publicuser"

		api := &Router{ds: ds}
		w := httptest.NewRecorder()
		api.getPublicUsers(w, httptest.NewRequest("GET", "/users/public", nil))
		Expect(w.Code).To(Equal(http.StatusOK))

		var users []dto.UserDto
		Expect(json.Unmarshal(w.Body.Bytes(), &users)).To(Succeed())
		Expect(users).To(HaveLen(1))
		Expect(users[0].PrimaryImageTag).To(Equal(u.AvatarTag()))
		Expect(users[0].PrimaryImageTag).ToNot(BeEmpty())
	})

	It("is empty on getPublicUsers when the listed user has no avatar", func() {
		ds := &tests.MockDataStore{}
		ur := ds.User(context.Background()).(*tests.MockedUserRepo)
		Expect(ur.Put(&model.User{ID: testID("pub1"), UserName: "publicuser"})).To(Succeed())
		conf.Server.Jellyfin.ExposedPublicUsers = "publicuser"

		api := &Router{ds: ds}
		w := httptest.NewRecorder()
		api.getPublicUsers(w, httptest.NewRequest("GET", "/users/public", nil))
		Expect(w.Code).To(Equal(http.StatusOK))

		var users []dto.UserDto
		Expect(json.Unmarshal(w.Body.Bytes(), &users)).To(Succeed())
		Expect(users).To(HaveLen(1))
		Expect(users[0].PrimaryImageTag).To(BeEmpty())
	})
})
