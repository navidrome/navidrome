package scanners

import "github.com/deluan/gosonic/models"

type Scanner interface {
	LoadFolder(path string) []models.MediaFile
}
