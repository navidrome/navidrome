package scanner

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/repositories"
	"github.com/deluan/gosonic/models"
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
	for _, t := range files {
		m := &models.MediaFile{
			Id: t.Id,
			Album: t.Album,
			Artist: t.Artist,
			Title: t.Title,
			Path: t.Path,
			CreatedAt: t.CreatedAt,
			UpdatedAt: t.UpdatedAt,
		}
		err := mfRepo.Add(m)
		if err != nil {
			beego.Error(err)
		}
	}
	mfRepo.Dump()
}