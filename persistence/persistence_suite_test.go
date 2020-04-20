package persistence

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/db"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/tests"
	_ "github.com/mattn/go-sqlite3"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPersistence(t *testing.T) {
	tests.Init(t, true)

	//os.Remove("./test-123.db")
	//conf.Server.Path = "./test-123.db"
	conf.Server.DbPath = "file::memory:?cache=shared"
	orm.RegisterDataBase("default", db.Driver, conf.Server.DbPath)
	New()
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
	albumSgtPeppers    = model.Album{ID: "101", Name: "Sgt Peppers", Artist: "The Beatles", AlbumArtistID: "3", Genre: "Rock", CoverArtId: "1", CoverArtPath: P("/beatles/1/sgt/a day.mp3"), SongCount: 1, MaxYear: 1967, FullText: " beatles peppers sgt the"}
	albumAbbeyRoad     = model.Album{ID: "102", Name: "Abbey Road", Artist: "The Beatles", AlbumArtistID: "3", Genre: "Rock", CoverArtId: "2", CoverArtPath: P("/beatles/1/come together.mp3"), SongCount: 1, MaxYear: 1969, FullText: " abbey beatles road the"}
	albumRadioactivity = model.Album{ID: "103", Name: "Radioactivity", Artist: "Kraftwerk", AlbumArtistID: "2", Genre: "Electronic", CoverArtId: "3", CoverArtPath: P("/kraft/radio/radio.mp3"), SongCount: 2, FullText: " kraftwerk radioactivity"}
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
		ID:      "10",
		Name:    "Best",
		Comment: "No Comments",
		Owner:   "userid",
		Public:  true,
		Tracks:  model.MediaFiles{{ID: "1001"}, {ID: "1003"}},
	}
	plsCool       = model.Playlist{ID: "11", Name: "Cool", Tracks: model.MediaFiles{{ID: "1004"}}}
	testPlaylists = model.Playlists{plsBest, plsCool}
)

func P(path string) string {
	return filepath.FromSlash(path)
}

var _ = Describe("Initialize test DB", func() {

	// TODO Load this data setup from file(s)
	BeforeSuite(func() {
		o := orm.NewOrm()
		ctx := context.WithValue(log.NewContext(nil), "user", model.User{ID: "userid"})
		mr := NewMediaFileRepository(ctx, o)
		for _, s := range testSongs {
			err := mr.Put(&s)
			if err != nil {
				panic(err)
			}
		}

		alr := NewAlbumRepository(ctx, o).(*albumRepository)
		for _, a := range testAlbums {
			_, err := alr.put(a.ID, &a)
			if err != nil {
				panic(err)
			}
		}

		arr := NewArtistRepository(ctx, o)
		for _, a := range testArtists {
			err := arr.Put(&a)
			if err != nil {
				panic(err)
			}
		}

		pr := NewPlaylistRepository(ctx, o)
		for _, pls := range testPlaylists {
			err := pr.Put(&pls)
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
