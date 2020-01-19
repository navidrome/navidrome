package persistence

import (
	"os"
	"strings"
	"testing"

	"github.com/cloudsonic/sonic-server/conf"
	"github.com/cloudsonic/sonic-server/log"
	"github.com/cloudsonic/sonic-server/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPersistence(t *testing.T) {
	log.SetLevel(log.LevelCritical)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Persistence Suite")
}

var artistSaaraSaara = model.Artist{ID: "1", Name: "Saara Saara", AlbumCount: 2}
var artistKraftwerk = model.Artist{ID: "2", Name: "Kraftwerk"}
var artistBeatles = model.Artist{ID: "3", Name: "The Beatles"}
var testArtists = model.Artists{
	artistSaaraSaara,
	artistKraftwerk,
	artistBeatles,
}

var albumSgtPeppers = model.Album{ID: "1", Name: "Sgt Peppers", Artist: "The Beatles", ArtistID: "1", Genre: "Rock"}
var albumAbbeyRoad = model.Album{ID: "2", Name: "Abbey Road", Artist: "The Beatles", ArtistID: "1", Genre: "Rock"}
var albumRadioactivity = model.Album{ID: "3", Name: "Radioactivity", Artist: "Kraftwerk", ArtistID: "2", Starred: true, Genre: "Electronic"}
var testAlbums = model.Albums{
	albumSgtPeppers,
	albumAbbeyRoad,
	albumRadioactivity,
}

var songDayInALife = model.MediaFile{ID: "1", Title: "A Day In A Life", ArtistID: "3", AlbumID: "1", Genre: "Rock", Path: P("/beatles/1/sgt/a day.mp3")}
var songComeTogether = model.MediaFile{ID: "2", Title: "Come Together", ArtistID: "3", AlbumID: "2", Genre: "Rock", Path: P("/beatles/1/come together.mp3")}
var songRadioactivity = model.MediaFile{ID: "3", Title: "Radioactivity", ArtistID: "2", AlbumID: "3", Genre: "Electronic", Path: P("/kraft/radio/radio.mp3")}
var songAntenna = model.MediaFile{ID: "4", Title: "Antenna", ArtistID: "2", AlbumID: "3", Genre: "Electronic", Path: P("/kraft/radio/antenna.mp3")}
var testSongs = model.MediaFiles{
	songDayInALife,
	songComeTogether,
	songRadioactivity,
	songAntenna,
}

func P(path string) string {
	return strings.ReplaceAll(path, "/", string(os.PathSeparator))
}

var _ = Describe("Initialize test DB", func() {
	BeforeSuite(func() {
		//log.SetLevel(log.LevelTrace)
		//conf.Sonic.DbPath, _ = ioutil.TempDir("", "cloudsonic_tests")
		//os.MkdirAll(conf.Sonic.DbPath, 0700)
		conf.Sonic.DbPath = ":memory:"
		ds := New()
		artistRepo := ds.Artist()
		for _, a := range testArtists {
			artistRepo.Put(&a)
		}
		albumRepository := ds.Album()
		for _, a := range testAlbums {
			err := albumRepository.Put(&a)
			if err != nil {
				panic(err)
			}
		}
		mediaFileRepository := ds.MediaFile()
		for _, s := range testSongs {
			err := mediaFileRepository.Put(&s, true)
			if err != nil {
				panic(err)
			}
		}
	})
})
