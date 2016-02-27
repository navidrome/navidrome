package scanner

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/repositories"
	"github.com/deluan/gosonic/models"
	"strings"
	"fmt"
	"encoding/json"
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
	updateDatastore(files)
	beego.Info("Finished importing", len(files), "files")
}

func updateDatastore(files []Track) {
	mfRepo := repositories.NewMediaFileRepository()
	var artistIndex = make(map[string]map[string]string)
	for _, t := range files {
		m := &models.MediaFile{
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
		err := mfRepo.Add(m)
		if err != nil {
			beego.Error(err)
		}
		collectIndex(m, artistIndex)
	}
	//mfRepo.Dump()
	j,_ := json.MarshalIndent(artistIndex, "", "    ")
	fmt.Println(string(j))
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