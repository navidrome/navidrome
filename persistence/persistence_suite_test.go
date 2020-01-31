package persistence

import (
	"os"
	"strings"
	"testing"

	"github.com/Masterminds/squirrel"
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
	//conf.Server.DbPath = "./test-123.db"
	conf.Server.DbPath = "file::memory:?cache=shared"
	New()
	db.EnsureDB()
	log.SetLevel(log.LevelCritical)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Persistence Suite")
}

var artistKraftwerk = model.Artist{ID: "2", Name: "Kraftwerk", AlbumCount: 1}
var artistBeatles = model.Artist{ID: "3", Name: "The Beatles", AlbumCount: 2}

var albumSgtPeppers = model.Album{ID: "1", Name: "Sgt Peppers", Artist: "The Beatles", ArtistID: "3", Genre: "Rock", CoverArtId: "1", CoverArtPath: P("/beatles/1/sgt/a day.mp3"), SongCount: 1}
var albumAbbeyRoad = model.Album{ID: "2", Name: "Abbey Road", Artist: "The Beatles", ArtistID: "3", Genre: "Rock", CoverArtId: "2", CoverArtPath: P("/beatles/1/come together.mp3"), SongCount: 1}
var albumRadioactivity = model.Album{ID: "3", Name: "Radioactivity", Artist: "Kraftwerk", ArtistID: "2", Genre: "Electronic", CoverArtId: "3", CoverArtPath: P("/kraft/radio/radio.mp3"), SongCount: 2, Starred: true}
var testAlbums = model.Albums{
	albumSgtPeppers,
	albumAbbeyRoad,
	albumRadioactivity,
}

var songDayInALife = model.MediaFile{ID: "1", Title: "A Day In A Life", ArtistID: "3", Artist: "The Beatles", AlbumID: "1", Album: "Sgt Peppers", Genre: "Rock", Path: P("/beatles/1/sgt/a day.mp3")}
var songComeTogether = model.MediaFile{ID: "2", Title: "Come Together", ArtistID: "3", Artist: "The Beatles", AlbumID: "2", Album: "Abbey Road", Genre: "Rock", Path: P("/beatles/1/come together.mp3"), Starred: true}
var songRadioactivity = model.MediaFile{ID: "3", Title: "Radioactivity", ArtistID: "2", Artist: "Kraftwerk", AlbumID: "3", Album: "Radioactivity", Genre: "Electronic", Path: P("/kraft/radio/radio.mp3")}
var songAntenna = model.MediaFile{ID: "4", Title: "Antenna", ArtistID: "2", Artist: "Kraftwerk", AlbumID: "3", Genre: "Electronic", Path: P("/kraft/radio/antenna.mp3")}
var testSongs = model.MediaFiles{
	songDayInALife,
	songComeTogether,
	songRadioactivity,
	songAntenna,
}

var annAlbumRadioactivity = model.Annotation{AnnID: "1", UserID: "userid", ItemType: model.AlbumItemType, ItemID: "3", Starred: true}
var annSongComeTogether = model.Annotation{AnnID: "2", UserID: "userid", ItemType: model.MediaItemType, ItemID: "2", Starred: true}
var testAnnotations = []model.Annotation{
	annAlbumRadioactivity,
	annSongComeTogether,
}

var (
	plsBest = model.Playlist{
		ID:       "10",
		Name:     "Best",
		Comment:  "No Comments",
		Duration: 10,
		Owner:    "userid",
		Public:   true,
		Tracks:   model.MediaFiles{{ID: "1"}, {ID: "3"}},
	}
	plsCool       = model.Playlist{ID: "11", Name: "Cool", Tracks: model.MediaFiles{{ID: "4"}}}
	testPlaylists = model.Playlists{plsBest, plsCool}
)

func P(path string) string {
	return strings.ReplaceAll(path, "/", string(os.PathSeparator))
}

var _ = Describe("Initialize test DB", func() {
	BeforeSuite(func() {
		o := orm.NewOrm()
		mr := NewMediaFileRepository(nil, o)
		for _, s := range testSongs {
			err := mr.Put(&s)
			if err != nil {
				panic(err)
			}
		}

		for _, a := range testAnnotations {
			values, _ := toSqlArgs(a)
			ins := squirrel.Insert("annotation").SetMap(values)
			query, args, err := ins.ToSql()
			if err != nil {
				panic(err)
			}
			_, err = o.Raw(query, args...).Exec()
			if err != nil {
				panic(err)
			}
		}

		pr := NewPlaylistRepository(nil, o)
		for _, pls := range testPlaylists {
			err := pr.Put(&pls)
			if err != nil {
				panic(err)
			}
		}
	})
})
