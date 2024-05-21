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
		_, err := db.Db().WriteDB().ExecContext(ctx, `
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
			revolver := template(_t{"albumartist": "The Beatles", "album": "Revolver", "year": 1966})
			help := template(_t{"albumartist": "The Beatles", "album": "Help!", "year": 1965})
			files = fstest.MapFS{
				"The Beatles/Revolver/01 - Taxman.mp3":                         revolver(track(1, "Taxman")),
				"The Beatles/Revolver/02 - Eleanor Rigby.mp3":                  revolver(track(2, "Eleanor Rigby")),
				"The Beatles/Revolver/03 - I'm Only Sleeping.mp3":              revolver(track(3, "I'm Only Sleeping")),
				"The Beatles/Help!/01 - Help!.mp3":                             help(track(1, "Help!")),
				"The Beatles/Help!/02 - The Night Before.mp3":                  help(track(2, "The Night Before")),
				"The Beatles/Help!/03 - You've Got to Hide Your Love Away.mp3": help(track(3, "You've Got to Hide Your Love Away")),
			}
		})

		It("should import all folders", func() {
			Expect(s.RescanAll(ctx, true)).To(Succeed())

			folders, _ := ds.Folder(ctx).GetAll(lib)
			paths := slice.Map(folders, func(f model.Folder) string { return f.Name })
			Expect(paths).To(SatisfyAll(
				HaveLen(4),
				ContainElements(".", "The Beatles", "Revolver", "Help!"),
			))
		})
		It("should import all mediafiles", func() {
			Expect(s.RescanAll(ctx, true)).To(Succeed())

			mfs, _ := ds.MediaFile(ctx).GetAll()
			paths := slice.Map(mfs, func(f model.MediaFile) string { return f.Title })
			Expect(paths).To(SatisfyAll(
				HaveLen(6),
				ContainElements(
					"Taxman", "Eleanor Rigby", "I'm Only Sleeping",
					"Help!", "The Night Before", "You've Got to Hide Your Love Away",
				),
			))
		})
	})
})
