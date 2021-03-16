package persistence

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/astaxie/beego/orm"
	_ "github.com/mattn/go-sqlite3"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPersistence(t *testing.T) {
	tests.Init(t, true)

	//os.Remove("./test-123.db")
	//conf.Server.DbPath = "./test-123.db"
	conf.Server.DbPath = "file::memory:?cache=shared"
	_ = orm.RegisterDataBase("default", db.Driver, conf.Server.DbPath)
	db.EnsureLatestVersion()
	log.SetLevel(log.LevelCritical)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Persistence Suite")
}

var (
	artistKraftwerk = model.Artist{ID: "2", Name: "Kraftwerk", AlbumCount: 1, FullText: " kraftwerk"}
	artistBeatles   = model.Artist{ID: "3", Name: "The Beatles", AlbumCount: 2, FullText: " beatles the"}
	testArtists     = model.Artists{
		artistKraftwerk,
		artistBeatles,
	}
)

var (
	albumSgtPeppers    = model.Album{ID: "101", Name: "Sgt Peppers", Artist: "The Beatles", OrderAlbumName: "sgt peppers", AlbumArtistID: "3", Genre: "Rock", CoverArtId: "1", CoverArtPath: P("/beatles/1/sgt/a day.mp3"), SongCount: 1, MaxYear: 1967, FullText: " beatles peppers sgt the"}
	albumAbbeyRoad     = model.Album{ID: "102", Name: "Abbey Road", Artist: "The Beatles", OrderAlbumName: "abbey road", AlbumArtistID: "3", Genre: "Rock", CoverArtId: "2", CoverArtPath: P("/beatles/1/come together.mp3"), SongCount: 1, MaxYear: 1969, FullText: " abbey beatles road the"}
	albumRadioactivity = model.Album{ID: "103", Name: "Radioactivity", Artist: "Kraftwerk", OrderAlbumName: "radioactivity", AlbumArtistID: "2", Genre: "Electronic", CoverArtId: "3", CoverArtPath: P("/kraft/radio/radio.mp3"), SongCount: 2, FullText: " kraftwerk radioactivity"}
	testAlbums         = model.Albums{
		albumSgtPeppers,
		albumAbbeyRoad,
		albumRadioactivity,
	}
)

var (
	songDayInALife    = model.MediaFile{ID: "1001", Title: "A Day In A Life", ArtistID: "3", Artist: "The Beatles", AlbumID: "101", Album: "Sgt Peppers", Genre: "Rock", Path: P("/beatles/1/sgt/a day.mp3"), FullText: " a beatles day in life peppers sgt the"}
	songComeTogether  = model.MediaFile{ID: "1002", Title: "Come Together", ArtistID: "3", Artist: "The Beatles", AlbumID: "102", Album: "Abbey Road", Genre: "Rock", Path: P("/beatles/1/come together.mp3"), FullText: " abbey beatles come road the together"}
	songRadioactivity = model.MediaFile{ID: "1003", Title: "Radioactivity", ArtistID: "2", Artist: "Kraftwerk", AlbumID: "103", Album: "Radioactivity", Genre: "Electronic", Path: P("/kraft/radio/radio.mp3"), FullText: " kraftwerk radioactivity"}
	songAntenna       = model.MediaFile{ID: "1004", Title: "Antenna", ArtistID: "2", Artist: "Kraftwerk", AlbumID: "103", Genre: "Electronic", Path: P("/kraft/radio/antenna.mp3"), FullText: " antenna kraftwerk"}
	testSongs         = model.MediaFiles{
		songDayInALife,
		songComeTogether,
		songRadioactivity,
		songAntenna,
	}
)

var (
	plsBest = model.Playlist{
		Name:      "Best",
		Comment:   "No Comments",
		Owner:     "userid",
		Public:    true,
		SongCount: 2,
		Tracks:    model.MediaFiles{{ID: "1001"}, {ID: "1003"}},
	}
	plsCool       = model.Playlist{Name: "Cool", Owner: "userid", Tracks: model.MediaFiles{{ID: "1004"}}}
	testPlaylists = []*model.Playlist{&plsBest, &plsCool}
)

func P(path string) string {
	return filepath.FromSlash(path)
}

var _ = Describe("Initialize test DB", func() {
	dirFS := os.DirFS(".")

	// TODO Load this data setup from file(s)
	BeforeSuite(func() {
		o := orm.NewOrm()
		ctx := log.NewContext(context.TODO())
		ctx = request.WithUser(ctx, model.User{ID: "userid", UserName: "userid"})
		mr := NewMediaFileRepository(ctx, o)
		for i := range testSongs {
			s := testSongs[i]
			err := mr.Put(&s)
			if err != nil {
				panic(err)
			}
		}

		alr := NewAlbumRepository(ctx, dirFS, o).(*albumRepository)
		for i := range testAlbums {
			a := testAlbums[i]
			_, err := alr.put(a.ID, &a)
			if err != nil {
				panic(err)
			}
		}

		arr := NewArtistRepository(ctx, o)
		for i := range testArtists {
			a := testArtists[i]
			err := arr.Put(&a)
			if err != nil {
				panic(err)
			}
		}

		pr := NewPlaylistRepository(ctx, o)
		for i := range testPlaylists {
			err := pr.Put(testPlaylists[i])
			if err != nil {
				panic(err)
			}
		}

		// Prepare annotations
		if err := arr.SetStar(true, artistBeatles.ID); err != nil {
			panic(err)
		}
		ar, _ := arr.Get(artistBeatles.ID)
		artistBeatles.Starred = true
		artistBeatles.StarredAt = ar.StarredAt
		testArtists[1] = artistBeatles

		if err := alr.SetStar(true, albumRadioactivity.ID); err != nil {
			panic(err)
		}
		al, _ := alr.Get(albumRadioactivity.ID)
		albumRadioactivity.Starred = true
		albumRadioactivity.StarredAt = al.StarredAt
		testAlbums[2] = albumRadioactivity

		if err := mr.SetStar(true, songComeTogether.ID); err != nil {
			panic(err)
		}
		mf, _ := mr.Get(songComeTogether.ID)
		songComeTogether.Starred = true
		songComeTogether.StarredAt = mf.StarredAt
		testSongs[1] = songComeTogether

	})
})
