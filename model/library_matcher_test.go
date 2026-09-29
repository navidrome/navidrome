package model_test

import (
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LibraryMatcher", func() {
	BeforeEach(func() {
		tests.SkipOnWindows("path separator bug (#TBD-path-sep-playlists)")
	})

	find := func(libs model.Libraries, path string) int {
		lib, ok := model.NewLibraryMatcher(libs).FindLibrary(path)
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
