package scanner

import (
	"fmt"
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/consts"
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/persistence"
	"github.com/deluan/gosonic/utils"
	"strings"
	"time"
	"github.com/dhowden/tag"
	"os"
)

type Scanner interface {
	LoadFolder(path string) []Track
}

type tempIndex map[string]domain.ArtistInfo

// TODO Implement a flag 'isScanning'.
func StartImport() {
	go doImport(beego.AppConfig.String("musicFolder"), &ItunesScanner{})
}

func doImport(mediaFolder string, scanner Scanner) {
	beego.Info("Starting iTunes import from:", mediaFolder)
	files := scanner.LoadFolder(mediaFolder)
	importLibrary(files)
	beego.Info("Finished importing", len(files), "files")
}

func importLibrary(files []Track) (err error) {
	indexGroups := utils.ParseIndexGroups(beego.AppConfig.String("indexGroups"))
	mfRepo := persistence.NewMediaFileRepository()
	albumRepo := persistence.NewAlbumRepository()
	artistRepo := persistence.NewArtistRepository()
	var artistIndex = make(map[string]tempIndex)

	for _, t := range files {
		mf, album, artist := parseTrack(&t)
		persist(mfRepo, mf, albumRepo, album, artistRepo, artist)
		collectIndex(indexGroups, artist, artistIndex)
	}

	if err = saveIndex(artistIndex); err != nil {
		beego.Error(err)
	}

	c, _ := artistRepo.CountAll()
	beego.Info("Total Artists in database:", c)
	c, _ = albumRepo.CountAll()
	beego.Info("Total Albums in database:", c)
	c, _ = mfRepo.CountAll()
	beego.Info("Total MediaFiles in database:", c)

	if err == nil {
		propertyRepo := persistence.NewPropertyRepository()
		millis := time.Now().UnixNano() / 1000000
		propertyRepo.Put(consts.LastScan, fmt.Sprint(millis))
		beego.Info("LastScan timestamp:", millis)
	}

	return err
}

func hasCoverArt(path string) bool {
	if _, err := os.Stat(path); err == nil {
		f, err := os.Open(path)
		if err != nil {
			beego.Warn("Error opening file", path, "-", err)
			return false
		}
		defer f.Close()

		m, err := tag.ReadFrom(f)
		if err != nil {
			beego.Warn("Error reading tag from file", path, "-", err)
		}
		return m.Picture() != nil
	}
	//beego.Warn("File not found:", path)
	return false
}

func parseTrack(t *Track) (*domain.MediaFile, *domain.Album, *domain.Artist) {
	hasCover := hasCoverArt(t.Path)
	mf := &domain.MediaFile{
		Id:          t.Id,
		Album:       t.Album,
		Artist:      t.Artist,
		AlbumArtist: t.AlbumArtist,
		Title:       t.Title,
		Compilation: t.Compilation,
		Starred:     t.Loved,
		Path:        t.Path,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		HasCoverArt: hasCover,
		TrackNumber: t.TrackNumber,
		DiscNumber:  t.DiscNumber,
		Genre:       t.Genre,
		Year:        t.Year,
		Size:        t.Size,
		Suffix:      t.Suffix,
		Duration:    t.Duration,
		BitRate:     t.BitRate,
	}

	album := &domain.Album{
		Name:        t.Album,
		Year:        t.Year,
		Compilation: t.Compilation,
		Starred:     t.AlbumLoved,
		Genre:       t.Genre,
		Artist:      t.Artist,
		AlbumArtist: t.AlbumArtist,
	}

	if mf.HasCoverArt {
		album.CoverArtId = mf.Id
	}

	artist := &domain.Artist{
		Name: t.RealArtist(),
	}

	return mf, album, artist
}

func persist(mfRepo domain.MediaFileRepository, mf *domain.MediaFile, albumRepo domain.AlbumRepository, album *domain.Album, artistRepo domain.ArtistRepository, artist *domain.Artist) {
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

func collectIndex(ig utils.IndexGroups, a *domain.Artist, artistIndex map[string]tempIndex) {
	name := a.Name
	indexName := strings.ToLower(utils.NoArticle(name))
	if indexName == "" {
		return
	}
	group := findGroup(ig, indexName)
	artists := artistIndex[group]
	if artists == nil {
		artists = make(tempIndex)
		artistIndex[group] = artists
	}
	artists[indexName] = domain.ArtistInfo{ArtistId: a.Id, Artist: a.Name}
}

func findGroup(ig utils.IndexGroups, name string) string {
	for k, v := range ig {
		key := strings.ToLower(k)
		if strings.HasPrefix(name, key) {
			return v
		}
	}
	return "#"
}

func saveIndex(artistIndex map[string]tempIndex) error {
	idxRepo := persistence.NewArtistIndexRepository()

	for k, temp := range artistIndex {
		idx := &domain.ArtistIndex{Id: k}
		for _, v := range temp {
			idx.Artists = append(idx.Artists, v)
		}
		err := idxRepo.Put(idx)
		if err != nil {
			return err
		}
	}

	return nil
}
