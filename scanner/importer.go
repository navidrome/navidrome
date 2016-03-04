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
)

type Scanner interface {
	ScanLibrary(path string) (int, error)
	MediaFiles() map[string]*domain.MediaFile
	Albums() map[string]*domain.Album
	Artists() map[string]*domain.Artist
}

type tempIndex map[string]domain.ArtistInfo

func StartImport() {
	go func() {
		i := &Importer{
			scanner:      &ItunesScanner{},
			mediaFolder:  beego.AppConfig.String("musicFolder"),
			mfRepo:       persistence.NewMediaFileRepository(),
			albumRepo:    persistence.NewAlbumRepository(),
			artistRepo:   persistence.NewArtistRepository(),
			idxRepo:      persistence.NewArtistIndexRepository(),
			propertyRepo: persistence.NewPropertyRepository(),
		}
		i.Run()
	}()
}

// TODO Implement a flag 'inProgress'.
type Importer struct {
	scanner      Scanner
	mediaFolder  string
	mfRepo       domain.MediaFileRepository
	albumRepo    domain.AlbumRepository
	artistRepo   domain.ArtistRepository
	idxRepo      domain.ArtistIndexRepository
	propertyRepo domain.PropertyRepository
}

func (i *Importer) Run() {
	beego.Info("Starting iTunes import from:", i.mediaFolder)
	if total, err := i.scanner.ScanLibrary(i.mediaFolder); err != nil {
		beego.Error("Error importing iTunes Library:", err)
		return
	} else {
		//fmt.Printf(">>>>>>>>>>>>>>>>>>\n%#v\n>>>>>>>>>>>>>>>>>\n", i.scanner.Albums())
		beego.Info("Found", total, "tracks,",
			len(i.scanner.MediaFiles()), "songs,",
			len(i.scanner.Albums()), "albums,",
			len(i.scanner.Artists()), "artists")
	}
	if err := i.importLibrary(); err != nil {
		beego.Error("Error persisting data:", err)
	}
	beego.Info("Finished importing tracks from iTunes Library")
}

func (i *Importer) importLibrary() (err error) {
	indexGroups := utils.ParseIndexGroups(beego.AppConfig.String("indexGroups"))
	artistIndex := make(map[string]tempIndex)

	for _, mf := range i.scanner.MediaFiles() {
		if err := i.mfRepo.Put(mf); err != nil {
			beego.Error(err)
		}
	}

	for _, al := range i.scanner.Albums() {
		if err := i.albumRepo.Put(al); err != nil {
			beego.Error(err)
		}
	}

	for _, ar := range i.scanner.Artists() {
		if err := i.artistRepo.Put(ar); err != nil {
			beego.Error(err)
		}
		i.collectIndex(indexGroups, ar, artistIndex)
	}

	if err = i.saveIndex(artistIndex); err != nil {
		beego.Error(err)
	}

	c, _ := i.artistRepo.CountAll()
	beego.Info("Total Artists in database:", c)
	c, _ = i.albumRepo.CountAll()
	beego.Info("Total Albums in database:", c)
	c, _ = i.mfRepo.CountAll()
	beego.Info("Total MediaFiles in database:", c)

	if err == nil {
		millis := time.Now().UnixNano() / 1000000
		i.propertyRepo.Put(consts.LastScan, fmt.Sprint(millis))
		beego.Info("LastScan timestamp:", millis)
	}

	return err
}

func (i *Importer) persist(mf *domain.MediaFile, album *domain.Album, artist *domain.Artist) {
	if err := i.artistRepo.Put(artist); err != nil {
		beego.Error(err)
	}

	album.ArtistId = artist.Id
	if err := i.albumRepo.Put(album); err != nil {
		beego.Error(err)
	}

	mf.AlbumId = album.Id
	if err := i.mfRepo.Put(mf); err != nil {
		beego.Error(err)
	}
}

func (i *Importer) collectIndex(ig utils.IndexGroups, a *domain.Artist, artistIndex map[string]tempIndex) {
	name := a.Name
	indexName := strings.ToLower(utils.NoArticle(name))
	if indexName == "" {
		return
	}
	group := i.findGroup(ig, indexName)
	artists := artistIndex[group]
	if artists == nil {
		artists = make(tempIndex)
		artistIndex[group] = artists
	}
	artists[indexName] = domain.ArtistInfo{ArtistId: a.Id, Artist: a.Name}
}

func (i *Importer) findGroup(ig utils.IndexGroups, name string) string {
	for k, v := range ig {
		key := strings.ToLower(k)
		if strings.HasPrefix(name, key) {
			return v
		}
	}
	return "#"
}

func (i *Importer) saveIndex(artistIndex map[string]tempIndex) error {
	for k, temp := range artistIndex {
		idx := &domain.ArtistIndex{Id: k}
		for _, v := range temp {
			idx.Artists = append(idx.Artists, v)
		}
		err := i.idxRepo.Put(idx)
		if err != nil {
			return err
		}
	}

	return nil
}
