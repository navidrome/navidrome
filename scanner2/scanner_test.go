package scanner2_test

import (
	"context"
	"testing/fstest"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/scanner"
	"github.com/navidrome/navidrome/scanner2"
	"github.com/navidrome/navidrome/utils/slice"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Scanner", func() {
	var fs *FakeFS
	var files fstest.MapFS
	var ctx context.Context
	var libRepo model.LibraryRepository
	var ds model.DataStore
	var s scanner.Scanner
	var lib model.Library

	BeforeEach(func() {
		log.SetLevel(log.LevelTrace)
		//os.Remove("./test-123.db")
		//conf.Server.DbPath = "./test-123.db"
		conf.Server.DbPath = "file::memory:?cache=shared"
		db.Init()

		ctx = context.Background()
		ds = persistence.New(db.Db())
		files = fstest.MapFS{}
		s = scanner2.GetInstance(ctx, ds)
	})

	AfterEach(func() {
		_, err := db.Db().ExecContext(ctx, `
			PRAGMA writable_schema = 1;
			DELETE FROM sqlite_master;
			PRAGMA writable_schema = 0;
			VACUUM;
			PRAGMA integrity_check;
		`)
		Expect(err).ToNot(HaveOccurred())
	})

	JustBeforeEach(func() {
		// Override the default library
		lib = model.Library{ID: 1, Name: "Fake Library", Path: "fake:///music"}
		libRepo = ds.Library(ctx)
		Expect(libRepo.Put(&lib)).To(Succeed())

		fs = &FakeFS{MapFS: files}
		RegisterFakeStorage(fs)
	})

	Describe("Scanner", func() {
		BeforeEach(func() {
			sgtPeppers := template(_t{"albumartist": "The Beatles", "album": "Sgt. Pepper's Lonely Hearts Club Band", "year": 1967})
			files = fstest.MapFS{
				"The Beatles/Sgt. Pepper's Lonely Hearts Club Band/01 - Sgt. Pepper's Lonely Hearts Club Band.mp3": sgtPeppers(track(1, "Sgt. Pepper's Lonely Hearts Club Band")),
				"The Beatles/Sgt. Pepper's Lonely Hearts Club Band/02 - With a Little Help from My Friends.mp3":    sgtPeppers(track(2, "With a Little Help from My Friends")),
				"The Beatles/Sgt. Pepper's Lonely Hearts Club Band/03 - Lucy in the Sky with Diamonds.mp3":         sgtPeppers(track(3, "Lucy in the Sky with Diamonds")),
				"The Beatles/Sgt. Pepper's Lonely Hearts Club Band/04 - Getting Better.mp3":                        sgtPeppers(track(4, "Getting Better")),
			}
		})

		It("should import all folders", func() {
			Expect(s.RescanAll(ctx, true)).To(Succeed())

			folders, _ := ds.Folder(ctx).GetAll(lib)
			paths := slice.Map(folders, func(f model.Folder) string { return f.Name })
			Expect(paths).To(SatisfyAll(
				HaveLen(3),
				ContainElements(".", "The Beatles", "Sgt. Pepper's Lonely Hearts Club Band"),
			))
		})
		It("should import all mediafiles", func() {
			Expect(s.RescanAll(ctx, true)).To(Succeed())

			mfs, _ := ds.MediaFile(ctx).GetAll()
			paths := slice.Map(mfs, func(f model.MediaFile) string { return f.Title })
			Expect(paths).To(SatisfyAll(
				HaveLen(4),
				ContainElements(
					"Sgt. Pepper's Lonely Hearts Club Band",
					"With a Little Help from My Friends",
					"Lucy in the Sky with Diamonds",
					"Getting Better",
				),
			))
		})
	})
})
