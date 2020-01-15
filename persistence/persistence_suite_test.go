package persistence

import (
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

var testAlbums = model.Albums{
	{ID: "1", Name: "Sgt Peppers", Artist: "The Beatles", ArtistID: "1"},
	{ID: "2", Name: "Abbey Road", Artist: "The Beatles", ArtistID: "1"},
	{ID: "3", Name: "Radioactivity", Artist: "Kraftwerk", ArtistID: "2", Starred: true},
}
var testArtists = model.Artists{
	{ID: "1", Name: "Saara Saara", AlbumCount: 2},
	{ID: "2", Name: "Kraftwerk"},
	{ID: "3", Name: "The Beatles"},
}

var _ = Describe("Initialize test DB", func() {
	BeforeSuite(func() {
		//conf.Sonic.DbPath, _ = ioutil.TempDir("", "cloudsonic_tests")
		//os.MkdirAll(conf.Sonic.DbPath, 0700)
		conf.Sonic.DbPath = ":memory:"
		Db()
		artistRepo := NewArtistRepository()
		for _, a := range testArtists {
			artistRepo.Put(&a)
		}
		albumRepository := NewAlbumRepository()
		for _, a := range testAlbums {
			err := albumRepository.Put(&a)
			if err != nil {
				panic(err)
			}
		}
	})
})
