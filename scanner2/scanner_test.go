package scanner2_test

import (
	"context"
	"testing/fstest"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/scanner"
	"github.com/navidrome/navidrome/scanner2"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Scanner", func() {
	var fs *FakeFS
	var files fstest.MapFS
	var ctx context.Context
	var libRepo tests.MockLibraryRepo
	var ds tests.MockDataStore
	var s scanner.Scanner
	var lib model.Library

	BeforeEach(func() {
		log.SetLevel(log.LevelTrace)
		ctx = context.Background()
		files = fstest.MapFS{}
		libRepo = tests.MockLibraryRepo{}
		ds.MockedLibrary = &libRepo
		s = scanner2.GetInstance(ctx, &ds)
	})

	JustBeforeEach(func() {
		libRepo.SetData(model.Libraries{lib})
		fs = &FakeFS{MapFS: files}
		RegisterFakeStorage(fs)
	})

	Describe("Scan", func() {
		BeforeEach(func() {
			lib = model.Library{Name: "Fake Library", Path: "fake:///music"}
			sgtPeppers := template(_t{"albumartist": "The Beatles", "album": "Sgt. Pepper's Lonely Hearts Club Band", "year": 1967})
			files = fstest.MapFS{
				"The Beatles/1967 - Sgt. Pepper's Lonely Hearts Club Band/cover.jpg":                                      file(),
				"The Beatles/1967 - Sgt. Pepper's Lonely Hearts Club Band/01 - Sgt. Pepper's Lonely Hearts Club Band.mp3": sgtPeppers(track(1, "Sgt. Pepper's Lonely Hearts Club Band")),
				"The Beatles/1967 - Sgt. Pepper's Lonely Hearts Club Band/02 - With a Little Help from My Friends.mp3":    sgtPeppers(track(2, "With a Little Help from My Friends")),
				"The Beatles/1967 - Sgt. Pepper's Lonely Hearts Club Band/03 - Lucy in the Sky with Diamonds.mp3":         sgtPeppers(track(3, "Lucy in the Sky with Diamonds")),
				"The Beatles/1967 - Sgt. Pepper's Lonely Hearts Club Band/04 - Getting Better.mp3":                        sgtPeppers(track(4, "Getting Better")),
			}
		})

		It("should scan all files", func() {
			err := s.RescanAll(context.Background(), true)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
