package jellyfin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
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
	var pub, priv *model.User

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

		priv = &model.User{ID: testID("u1"), UserName: "alice"}
		Expect(ur.Put(priv)).To(Succeed())

		api = New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
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
		tok, err := auth.CreateToken(priv)
		Expect(err).ToNot(HaveOccurred())
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/userimage?userId="+dto.EncodeID(priv.ID), nil)
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
