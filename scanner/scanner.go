package scanner

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/repositories"
	"github.com/deluan/gosonic/models"
	"strings"
"github.com/deluan/gosonic/utils"
)

type Scanner interface {
	LoadFolder(path string) []Track
}

type tempIndex map[string]*models.ArtistInfo

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
	var artistIndex = make(map[string]tempIndex)

	for _, t := range files {
		mf, album, artist := parseTrack(&t)
		persist(mfRepo, mf, albumRepo, album, artistRepo, artist)
		collectIndex(artist, artistIndex)
	}

	if err := saveIndex(artistIndex); err != nil {
		beego.Error(err)
	}

	c, _ := artistRepo.CountAll()
	beego.Info("Total Artists in database:", c)
	c, _ = albumRepo.CountAll()
	beego.Info("Total Albums in database:", c)
	c, _ = mfRepo.CountAll()
	beego.Info("Total MediaFiles in database:", c)
}

func parseTrack(t *Track) (*models.MediaFile, *models.Album, *models.Artist) {
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
		Name: t.RealArtist(),
	}

	return mf, album, artist
}

func persist(mfRepo *repositories.MediaFile, mf *models.MediaFile, albumRepo *repositories.Album, album *models.Album, artistRepo *repositories.Artist, artist *models.Artist) {
	if err := artistRepo.Put(artist); err != nil {
		beego.Error(err)
	}

	album.ArtistId = artist.Id
	if err := albumRepo.Put(album); err != nil {
		beego.Error(err)
	}

	mf.AlbumId = album.Id
	if err := mfRepo.Put(mf); err != nil {
		beego.Error(err)
	}
}

func collectIndex(a *models.Artist, artistIndex map[string]tempIndex) {
	name := a.Name
	indexName := strings.ToLower(utils.NoArticle(name))
	if indexName == "" {
		return
	}
	initial := strings.ToUpper(indexName[0:1])
	artists := artistIndex[initial]
	if artists == nil {
		artists = make(tempIndex)
		artistIndex[initial] = artists
	}
	artists[indexName] = &models.ArtistInfo{ArtistId: a.Id, Artist: a.Name}
}

func saveIndex(artistIndex map[string]tempIndex) error {
	idxRepo := repositories.NewArtistIndexRepository()

	for k, temp := range artistIndex {
		idx := &models.ArtistIndex{Id: k}
		for _, v := range temp {
			idx.Artists = append(idx.Artists, *v)
		}
		err := idxRepo.Put(idx)
		if err != nil {
			return err
		}
	}

	return nil
}