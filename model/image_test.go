package model_test

import (
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("UploadedImagePath", func() {
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.DataFolder = conf.NewDir("/data")
	})

	It("puts user avatars in the avatar folder, not under artwork", func() {
		Expect(model.UploadedImagePath(consts.EntityUser, "abc_deluan.png")).
			To(Equal(filepath.Join("/data", "avatar", "abc_deluan.png")))
	})

	It("keeps other entities under the artwork folder", func() {
		Expect(model.UploadedImagePath(consts.EntityArtist, "abc_bowie.png")).
			To(Equal(filepath.Join("/data", "artwork", "artist", "abc_bowie.png")))
	})
})
