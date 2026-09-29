package cmd

import (
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("inspect", func() {
	Describe("libraryForFile", func() {
		var matcher *model.LibraryMatcher
		var cwd string

		BeforeEach(func() {
			tests.SkipOnWindows("path separator bug (#TBD-path-sep-playlists)")
			var err error
			cwd, err = os.Getwd()
			Expect(err).ToNot(HaveOccurred())
			matcher = model.NewLibraryMatcher(model.Libraries{
				{ID: 1, Path: "/music"},
				{ID: 2, Path: filepath.Join(cwd, "loose"), PIDAlbum: "folder"},
			})
		})

		It("returns the library that contains an absolute path", func() {
			lib, ok := libraryForFile(matcher, "/music/album/track.mp3")
			Expect(ok).To(BeTrue())
			Expect(lib.ID).To(Equal(1))
		})

		It("resolves a relative path against the working directory", func() {
			lib, ok := libraryForFile(matcher, "loose/track.mp3")
			Expect(ok).To(BeTrue())
			Expect(lib.PIDAlbum).To(Equal("folder"))
		})

		It("falls back to the default library without overrides", func() {
			lib, ok := libraryForFile(matcher, "/elsewhere/track.mp3")
			Expect(ok).To(BeFalse())
			Expect(lib).To(Equal(model.Library{ID: model.DefaultLibraryID}))
		})
	})

	Describe("loadLibraries", func() {
		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
		})

		It("does not create a database when there is none", func() {
			dbFile := filepath.Join(GinkgoT().TempDir(), "navidrome.db")
			conf.Server.DbPath = dbFile + "?_journal_mode=WAL"

			Expect(loadLibraries(GinkgoT().Context())).To(BeNil())
			Expect(dbFile).ToNot(BeAnExistingFile())
		})
	})
})
