package scanner

import (
	"fmt"
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/consts"
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/persistence"
	"github.com/deluan/gosonic/utils"
	"github.com/dhowden/tag"
	"os"
	"strings"
	"time"
)

type Scanner interface {
	LoadFolder(path string) []Track
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
	files := i.scanner.LoadFolder(i.mediaFolder)
	i.importLibrary(files)
	beego.Info("Finished importing", len(files), "files")
}

func (i *Importer) importLibrary(files []Track) (err error) {
	indexGroups := utils.ParseIndexGroups(beego.AppConfig.String("indexGroups"))
	var artistIndex = make(map[string]tempIndex)

	for _, t := range files {
		mf, album, artist := i.parseTrack(&t)
		i.persist(mf, album, artist)
		i.collectIndex(indexGroups, artist, artistIndex)
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

func (i *Importer) hasCoverArt(path string) bool {
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

func (i *Importer) parseTrack(t *Track) (*domain.MediaFile, *domain.Album, *domain.Artist) {
	hasCover := i.hasCoverArt(t.Path)
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
		CreatedAt:   t.CreatedAt, // TODO Collect all songs for an album first
		UpdatedAt:   t.UpdatedAt,
	}

	if mf.HasCoverArt {
		album.CoverArtId = mf.Id
	}

	artist := &domain.Artist{
		Name: t.RealArtist(),
	}

	return mf, album, artist
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
