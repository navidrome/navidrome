package scanner

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/repositories"
	"github.com/deluan/gosonic/models"
	"strings"
	"fmt"
)

type Scanner interface {
	LoadFolder(path string) []Track
}

func StartImport() {
	go doImport(beego.AppConfig.String("musicFolder"), &ItunesScanner{})
}

func doImport(mediaFolder string, scanner Scanner) {
	beego.Info("Starting iTunes import from:", mediaFolder)
	files := scanner.LoadFolder(mediaFolder)
	importLibrary(files)
	beego.Info("Finished importing", len(files), "files")
}

func importLibrary(files []Track) {
	mfRepo := repositories.NewMediaFileRepository()
	albumRepo := repositories.NewAlbumRepository()
	artistRepo := repositories.NewArtistRepository()
	var artistIndex = make(map[string]map[string]string)

	for _, t := range files {
		mf, album, artist := processTrack(&t)
		mergeInfo(mfRepo, mf, albumRepo, album, artistRepo, artist)
		fmt.Printf("%#v\n", album)
		fmt.Printf("%#v\n\n", artist)
		collectIndex(mf, artistIndex)
	}

	//j,_ := json.MarshalIndent(artistIndex, "", "    ")
	//fmt.Println(string(j))
	c, _ := mfRepo.CountAll()

	beego.Info("Total mediafiles in database:", c)
}

func processTrack(t *Track) (*models.MediaFile, *models.Album, *models.Artist) {
	mf := &models.MediaFile{
		Id: t.Id,
		Album: t.Album,
		Artist: t.Artist,
		AlbumArtist: t.AlbumArtist,
		Title: t.Title,
		Compilation: t.Compilation,
		Path: t.Path,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}

	album := &models.Album{
		Name: t.Album,
		Year: t.Year,
		Compilation: t.Compilation,
	}

	artist := &models.Artist{
		Name: t.Artist,
	}

	return mf, album, artist
}

func mergeInfo(mfRepo *repositories.MediaFile, mf *models.MediaFile, albumRepo *repositories.Album, album *models.Album, artistRepo *repositories.Artist, artist *models.Artist) {
	artist.Id = artistRepo.NewId(artist.Name)

	sAlbum, err := albumRepo.GetByName(album.Name)
	if err != nil {
		beego.Error(err)
	}
	album.ArtistId = artist.Id
	album.AddMediaFiles(mf, sAlbum.MediaFiles)
	sAlbum, err = albumRepo.Put(album)
	if err != nil {
		beego.Error(err)
	}

	sArtist, err := artistRepo.GetByName(artist.Name)
	if err != nil {
		beego.Error(err)
	}
	artist.AddAlbums(sAlbum, sArtist.Albums)
	sArtist, err = artistRepo.Put(artist)
	if err != nil {
		beego.Error(err)
	}

	if err := mfRepo.Put(mf); err != nil {
		beego.Error(err)
	}
}

func collectIndex(m *models.MediaFile, artistIndex map[string]map[string]string) {
	name := m.RealArtist()
	indexName := strings.ToLower(models.NoArticle(name))
	if indexName == "" {
		return
	}
	initial := strings.ToUpper(indexName[0:1])
	artists := artistIndex[initial]
	if artists == nil {
		artists = make(map[string]string)
		artistIndex[initial] = artists
	}
	artists[indexName] = name
}