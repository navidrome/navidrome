package storm

import (
	"testing"

	"github.com/cloudsonic/sonic-server/domain"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestStormPersistence(t *testing.T) {
	//log.SetLevel(log.LevelCritical)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Storm Persistence Suite")
}

var testAlbums = domain.Albums{
	{ID: "1", Name: "Sgt Peppers", Artist: "The Beatles", ArtistID: "1"},
	{ID: "2", Name: "Abbey Road", Artist: "The Beatles", ArtistID: "1"},
	{ID: "3", Name: "Radioactivity", Artist: "Kraftwerk", ArtistID: "2", Starred: true},
}
var testArtists = domain.Artists{
	{ID: "1", Name: "Saara Saara"},
	{ID: "2", Name: "Kraftwerk"},
	{ID: "3", Name: "The Beatles"},
}

var _ = Describe("Initialize test DB", func() {
	BeforeSuite(func() {
		Db().Drop(&_Album{})
		albumRepo := NewAlbumRepository()
		for _, a := range testAlbums {
			albumRepo.Put(&a)
		}

		Db().Drop(&_Artist{})
		artistRepo := NewArtistRepository()
		for _, a := range testArtists {
			artistRepo.Put(&a)
		}
	})

})
