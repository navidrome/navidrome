package scanner2_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/storage/storagetest"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/scanner2"
)

func BenchmarkScan(b *testing.B) {
	tmpDir := os.TempDir()
	conf.Server.DbPath = filepath.Join(tmpDir, "test-scanner.db?_journal_mode=WAL")
	db.Init()
	ds := persistence.New(db.Db())
	s := scanner2.GetInstance(context.Background(), ds)

	fs := storagetest.FakeFS{}
	storagetest.Register("fake", &fs)
	var beatlesMBID = uuid.NewString()
	beatles := _t{
		"artist":                    "The Beatles",
		"artistsort":                "Beatles, The",
		"musicbrainz_artistid":      beatlesMBID,
		"albumartist":               "The Beatles",
		"albumartistsort":           "Beatles The",
		"musicbrainz_albumartistid": beatlesMBID,
	}
	revolver := template(beatles, _t{"album": "Revolver", "year": 1966, "composer": "Lennon/McCartney"})
	help := template(beatles, _t{"album": "Help!", "year": 1965, "composer": "Lennon/McCartney"})
	fs.SetFiles(fstest.MapFS{
		"The Beatles/Revolver/01 - Taxman.mp3":                         revolver(track(1, "Taxman")),
		"The Beatles/Revolver/02 - Eleanor Rigby.mp3":                  revolver(track(2, "Eleanor Rigby")),
		"The Beatles/Revolver/03 - I'm Only Sleeping.mp3":              revolver(track(3, "I'm Only Sleeping")),
		"The Beatles/Revolver/04 - Love You To.mp3":                    revolver(track(4, "Love You To")),
		"The Beatles/Help!/01 - Help!.mp3":                             help(track(1, "Help!")),
		"The Beatles/Help!/02 - The Night Before.mp3":                  help(track(2, "The Night Before")),
		"The Beatles/Help!/03 - You've Got to Hide Your Love Away.mp3": help(track(3, "You've Got to Hide Your Love Away")),
	})

	lib := model.Library{ID: 1, Name: "Fake Library", Path: "fake:///music"}
	ds.Library(context.Background()).Put(&lib)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := s.RescanAll(context.Background(), true)
		if err != nil {
			b.Fatal(err)
		}
	}
}
