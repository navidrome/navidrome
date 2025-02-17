package scanner_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"testing/fstest"

	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/metrics"
	"github.com/navidrome/navidrome/core/storage/storagetest"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/scanner"
	"github.com/navidrome/navidrome/server/events"
	"go.uber.org/goleak"
)

func BenchmarkScan(b *testing.B) {
	// Detect any goroutine leaks in the scanner code under test
	defer goleak.VerifyNone(b,
		goleak.IgnoreTopFunction("testing.(*B).run1"),
		goleak.IgnoreAnyFunction("testing.(*B).doBench"),
		// Ignore database/sql.(*DB).connectionOpener, as we are not closing the database connection
		goleak.IgnoreAnyFunction("database/sql.(*DB).connectionOpener"),
	)

	tmpDir := os.TempDir()
	conf.Server.DbPath = filepath.Join(tmpDir, "test-scanner.db?_journal_mode=WAL")
	db.Init(context.Background())

	ds := persistence.New(db.Db())
	conf.Server.DevExternalScanner = false
	s := scanner.New(context.Background(), ds, artwork.NoopCacheWarmer(), events.NoopBroker(),
		core.NewPlaylists(ds), metrics.NewNoopInstance())

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
	err := ds.Library(context.Background()).Put(&lib)
	if err != nil {
		b.Fatal(err)
	}

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.ScanAll(context.Background(), true)
		if err != nil {
			b.Fatal(err)
		}
	}

	runtime.ReadMemStats(&m2)
	fmt.Println("total:", humanize.Bytes(m2.TotalAlloc-m1.TotalAlloc))
	fmt.Println("mallocs:", humanize.Comma(int64(m2.Mallocs-m1.Mallocs)))
}
