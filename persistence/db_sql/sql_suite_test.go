package db_sql

import (
	"testing"

	"github.com/cloudsonic/sonic-server/conf"
	"github.com/cloudsonic/sonic-server/domain"
	"github.com/cloudsonic/sonic-server/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSQLitePersistence(t *testing.T) {
	log.SetLevel(log.LevelDebug)
	RegisterFailHandler(Fail)
	RunSpecs(t, "SQLite Persistence Suite")
}

var testAlbums = domain.Albums{
	{ID: "1", Name: "Sgt Peppers", Artist: "The Beatles", ArtistID: "1"},
	{ID: "2", Name: "Abbey Road", Artist: "The Beatles", ArtistID: "1"},
	{ID: "3", Name: "Radioactivity", Artist: "Kraftwerk", ArtistID: "2", Starred: true},
}
var testArtists = domain.Artists{
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
