package persistence

import (
	"context"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPersistence(t *testing.T) {
	tests.Init(t, true)

	//os.Remove("./test-123.db")
	//conf.Server.DbPath = "./test-123.db"
	conf.Server.DbPath = "file::memory:?cache=shared&_foreign_keys=on"
	defer db.Init()()
	log.SetLevel(log.LevelError)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Persistence Suite")
}

//var (
//	genreElectronic = model.Genre{ID: "gn-1", Name: "Electronic"}
//	genreRock       = model.Genre{ID: "gn-2", Name: "Rock"}
//	testGenres      = model.Genres{genreElectronic, genreRock}
//)

var (
	artistKraftwerk = model.Artist{ID: "2", Name: "Kraftwerk", AlbumCount: 1}
	artistBeatles   = model.Artist{ID: "3", Name: "The Beatles", AlbumCount: 2}
	testArtists     = model.Artists{
		artistKraftwerk,
		artistBeatles,
	}
)

var (
	albumSgtPeppers    = model.Album{LibraryID: 1, ID: "101", Name: "Sgt Peppers", Artist: "The Beatles", OrderAlbumName: "sgt peppers", AlbumArtistID: "3", EmbedArtPath: P("/beatles/1/sgt/a day.mp3"), SongCount: 1, MaxYear: 1967, Discs: model.Discs{}}
	albumAbbeyRoad     = model.Album{LibraryID: 1, ID: "102", Name: "Abbey Road", Artist: "The Beatles", OrderAlbumName: "abbey road", AlbumArtistID: "3", EmbedArtPath: P("/beatles/1/come together.mp3"), SongCount: 1, MaxYear: 1969, Discs: model.Discs{}}
	albumRadioactivity = model.Album{LibraryID: 1, ID: "103", Name: "Radioactivity", Artist: "Kraftwerk", OrderAlbumName: "radioactivity", AlbumArtistID: "2", EmbedArtPath: P("/kraft/radio/radio.mp3"), SongCount: 2, Discs: model.Discs{}}
	testAlbums         = model.Albums{
		albumSgtPeppers,
		albumAbbeyRoad,
		albumRadioactivity,
	}
)

var (
	songDayInALife    = model.MediaFile{LibraryID: 1, ID: "1001", Title: "A Day In A Life", ArtistID: "3", Artist: "The Beatles", AlbumID: "101", Album: "Sgt Peppers", Path: P("/beatles/1/sgt/a day.mp3")}
	songComeTogether  = model.MediaFile{LibraryID: 1, ID: "1002", Title: "Come Together", ArtistID: "3", Artist: "The Beatles", AlbumID: "102", Album: "Abbey Road", Path: P("/beatles/1/come together.mp3")}
	songRadioactivity = model.MediaFile{LibraryID: 1, ID: "1003", Title: "Radioactivity", ArtistID: "2", Artist: "Kraftwerk", AlbumID: "103", Album: "Radioactivity", Path: P("/kraft/radio/radio.mp3")}
	songAntenna       = model.MediaFile{LibraryID: 1, ID: "1004", Title: "Antenna", ArtistID: "2", Artist: "Kraftwerk",
		AlbumID:     "103",
		Path:        P("/kraft/radio/antenna.mp3"),
		RgAlbumGain: 1.0, RgAlbumPeak: 2.0, RgTrackGain: 3.0, RgTrackPeak: 4.0,
	}
	testSongs = model.MediaFiles{
		songDayInALife,
		songComeTogether,
		songRadioactivity,
		songAntenna,
	}
)

var (
	radioWithoutHomePage = model.Radio{ID: "1235", StreamUrl: "https://example.com:8000/1/stream.mp3", HomePageUrl: "", Name: "No Homepage"}
	radioWithHomePage    = model.Radio{ID: "5010", StreamUrl: "https://example.com/stream.mp3", Name: "Example Radio", HomePageUrl: "https://example.com"}
	testRadios           = model.Radios{radioWithoutHomePage, radioWithHomePage}
)

var (
	plsBest       model.Playlist
	plsCool       model.Playlist
	testPlaylists []*model.Playlist
)

var (
	adminUser   = model.User{ID: "userid", UserName: "userid", Name: "admin", Email: "admin@email.com", IsAdmin: true}
	regularUser = model.User{ID: "2222", UserName: "regular-user", Name: "Regular User", Email: "regular@example.com"}
	testUsers   = model.Users{adminUser, regularUser}
)

func P(path string) string {
	return filepath.FromSlash(path)
}

// Initialize test DB
// TODO Load this data setup from file(s)
var _ = BeforeSuite(func() {
	conn := NewDBXBuilder(db.Db())
	ctx := log.NewContext(context.TODO())
	ctx = request.WithUser(ctx, adminUser)

	ur := NewUserRepository(ctx, conn)
	for i := range testUsers {
		err := ur.Put(&testUsers[i])
		if err != nil {
			panic(err)
		}
	}

	//gr := NewGenreRepository(ctx, conn)
	//for i := range testGenres {
	//	g := testGenres[i]
	//	err := gr.Put(&g)
	//	if err != nil {
	//		panic(err)
	//	}
	//}

	mr := NewMediaFileRepository(ctx, conn)
	for i := range testSongs {
		err := mr.Put(&testSongs[i])
		if err != nil {
			panic(err)
		}
	}

	alr := NewAlbumRepository(ctx, conn).(*albumRepository)
	for i := range testAlbums {
		a := testAlbums[i]
		err := alr.Put(&a)
		if err != nil {
			panic(err)
		}
	}

	arr := NewArtistRepository(ctx, conn)
	for i := range testArtists {
		a := testArtists[i]
		err := arr.Put(&a)
		if err != nil {
			panic(err)
		}
	}

	rar := NewRadioRepository(ctx, conn)
	for i := range testRadios {
		r := testRadios[i]
		err := rar.Put(&r)
		if err != nil {
			panic(err)
		}
	}

	plsBest = model.Playlist{
		Name:      "Best",
		Comment:   "No Comments",
		OwnerID:   "userid",
		OwnerName: "userid",
		Public:    true,
		SongCount: 2,
	}
	plsBest.AddTracks([]string{"1001", "1003"})
	plsCool = model.Playlist{Name: "Cool", OwnerID: "userid", OwnerName: "userid"}
	plsCool.AddTracks([]string{"1004"})
	testPlaylists = []*model.Playlist{&plsBest, &plsCool}

	pr := NewPlaylistRepository(ctx, conn)
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
	al, err := alr.Get(albumRadioactivity.ID)
	if err != nil {
		panic(err)
	}
	albumRadioactivity.Starred = true
	albumRadioactivity.StarredAt = al.StarredAt
	testAlbums[2] = albumRadioactivity

	if err := mr.SetStar(true, songComeTogether.ID); err != nil {
		panic(err)
	}
	mf, err := mr.Get(songComeTogether.ID)
	if err != nil {
		panic(err)
	}
	songComeTogether.Starred = true
	songComeTogether.StarredAt = mf.StarredAt
	testSongs[1] = songComeTogether
})
