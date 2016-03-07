package engine

import (
	"errors"
	"fmt"
	"github.com/deluan/gosonic/consts"
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/utils"
	"strconv"
	"time"
)

type Browser interface {
	MediaFolders() (domain.MediaFolders, error)
	Indexes(ifModifiedSince time.Time) (domain.ArtistIndexes, time.Time, error)
}

func NewBrowser(propRepo domain.PropertyRepository, folderRepo domain.MediaFolderRepository, indexRepo domain.ArtistIndexRepository) Browser {
	return browser{propRepo, folderRepo, indexRepo}
}

type browser struct {
	propRepo   domain.PropertyRepository
	folderRepo domain.MediaFolderRepository
	indexRepo  domain.ArtistIndexRepository
}

func (b browser) MediaFolders() (domain.MediaFolders, error) {
	return b.folderRepo.GetAll()
}

func (b browser) Indexes(ifModifiedSince time.Time) (domain.ArtistIndexes, time.Time, error) {
	l, err := b.propRepo.DefaultGet(consts.LastScan, "-1")
	ms, _ := strconv.ParseInt(l, 10, 64)
	lastModified := utils.ToTime(ms)

	if err != nil {
		return domain.ArtistIndexes{}, time.Time{}, errors.New(fmt.Sprintf("Error retrieving LastScan property: %v", err))
	}

	if lastModified.After(ifModifiedSince) {
		indexes, err := b.indexRepo.GetAll()
		return indexes, lastModified, err
	}

	return domain.ArtistIndexes{}, lastModified, nil
}
