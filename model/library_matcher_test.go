package model_test

import (
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LibraryMatcher", func() {
	// Paths are written Unix-style and converted, so they use the OS separator, as filepath.Abs output does
	find := func(libs model.Libraries, path string) int {
		for i := range libs {
			libs[i].Path = filepath.FromSlash(libs[i].Path)
		}
		lib, ok := model.NewLibraryMatcher(libs).FindLibrary(filepath.FromSlash(path))
		if !ok {
			return 0
		}
		return lib.ID
	}

	DescribeTable("matches the longest library path",
		func(libs model.Libraries, path string, expectedID int) {
			Expect(find(libs, path)).To(Equal(expectedID))
		},
		Entry("nested library", model.Libraries{{ID: 1, Path: "/music"}, {ID: 2, Path: "/music-classical"}, {ID: 3, Path: "/music-classical/opera"}}, "/music-classical/opera/subdir/track.mp3", 3),
		Entry("sibling with a shared prefix", model.Libraries{{ID: 1, Path: "/music"}, {ID: 2, Path: "/music-classical"}}, "/music-classical/track.mp3", 2),
		Entry("shorter library", model.Libraries{{ID: 1, Path: "/music"}, {ID: 2, Path: "/music-classical"}}, "/music/track.mp3", 1),
		Entry("exact library root", model.Libraries{{ID: 1, Path: "/music"}, {ID: 2, Path: "/music-classical"}}, "/music-classical", 2),
		Entry("deeply nested libraries", model.Libraries{{ID: 1, Path: "/media"}, {ID: 2, Path: "/media/audio"}, {ID: 3, Path: "/media/audio/classical"}, {ID: 4, Path: "/media/audio/classical/baroque"}}, "/media/audio/classical/mozart/track.mp3", 3),
		Entry("prefix that is not a path boundary", model.Libraries{{ID: 1, Path: "/a"}, {ID: 2, Path: "/ab"}, {ID: 3, Path: "/abc"}}, "/ab/file.mp3", 2),
		Entry("special characters match literally", model.Libraries{{ID: 1, Path: "/music[test]"}, {ID: 2, Path: "/music(backup)"}}, "/music[test]/track.mp3", 1),
		Entry("library path with a trailing slash", model.Libraries{{ID: 1, Path: "/music/"}}, "/music/track.mp3", 1),
		Entry("library at the filesystem root", model.Libraries{{ID: 1, Path: "/"}}, "/music/track.mp3", 1),
		Entry("nested library under a root library", model.Libraries{{ID: 1, Path: "/"}, {ID: 2, Path: "/music"}}, "/music/track.mp3", 2),
	)

	It("does not match a path outside every library", func() {
		Expect(find(model.Libraries{{ID: 1, Path: "/music"}}, "/music-backup/track.mp3")).To(BeZero())
	})

	It("does not match anything without libraries", func() {
		Expect(find(nil, "/music/track.mp3")).To(BeZero())
	})

	It("does not reorder the caller's libraries", func() {
		libs := model.Libraries{{ID: 1, Path: "/a"}, {ID: 2, Path: "/abc"}}
		model.NewLibraryMatcher(libs)
		Expect(libs.IDs()).To(Equal([]int{1, 2}))
	})
})

var _ = Describe("LibraryRelativePath", func() {
	// Paths are built with filepath so the "absolute" cases stay absolute on every OS
	// (a Unix-style "/foo" is not absolute on Windows).
	libRoot, _ := filepath.Abs(filepath.Join("jukebox", "collection"))
	outside, _ := filepath.Abs(filepath.Join("somewhere", "else"))

	It("returns a relative path unchanged", func() {
		Expect(model.LibraryRelativePath(libRoot, "_Collection")).To(Equal("_Collection"))
	})

	It("rebases an absolute target when the library root is relative", func() {
		cwd, err := os.Getwd()
		Expect(err).ToNot(HaveOccurred())
		Expect(model.LibraryRelativePath(filepath.Join("music", "library"), filepath.Join(cwd, "music", "library", "rock"))).To(Equal("rock"))
	})

	It("rebases an absolute path that equals the library root to '.'", func() {
		Expect(model.LibraryRelativePath(libRoot, libRoot)).To(Equal("."))
	})

	It("rebases an absolute path under the library root", func() {
		Expect(model.LibraryRelativePath(libRoot, filepath.Join(libRoot, "_Collection"))).To(Equal("_Collection"))
	})

	It("handles a trailing slash on the library path", func() {
		Expect(model.LibraryRelativePath(libRoot+string(filepath.Separator), filepath.Join(libRoot, "_Collection"))).To(Equal("_Collection"))
	})

	It("leaves an absolute path outside the library root unchanged", func() {
		Expect(model.LibraryRelativePath(libRoot, outside)).To(Equal(outside))
	})

	It("returns an empty path unchanged", func() {
		Expect(model.LibraryRelativePath(libRoot, "")).To(Equal(""))
	})
})
