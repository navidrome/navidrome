package scanner

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/consts"
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/engine"
	"github.com/deluan/gosonic/utils"
)

type Scanner interface {
	ScanLibrary(lastModifiedSince time.Time, path string) (int, error)
	MediaFiles() map[string]*domain.MediaFile
	Albums() map[string]*domain.Album
	Artists() map[string]*domain.Artist
	Playlists() map[string]*domain.Playlist
}

type tempIndex map[string]domain.ArtistInfo

func StartImport() {
	go func() {
		// TODO Move all to DI
		i := &Importer{mediaFolder: beego.AppConfig.String("musicFolder")}
		utils.ResolveDependency(&i.mfRepo)
		utils.ResolveDependency(&i.albumRepo)
		utils.ResolveDependency(&i.artistRepo)
		utils.ResolveDependency(&i.idxRepo)
		utils.ResolveDependency(&i.plsRepo)
		utils.ResolveDependency(&i.propertyRepo)
		utils.ResolveDependency(&i.search)
		utils.ResolveDependency(&i.scanner)
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
	plsRepo      domain.PlaylistRepository
	propertyRepo engine.PropertyRepository
	search       engine.Search
	lastScan     time.Time
}

func (i *Importer) Run() {
	i.lastScan = i.lastModifiedSince()

	total, err := i.scanner.ScanLibrary(i.lastScan, i.mediaFolder)
	if err != nil {
		beego.Error("Error importing iTunes Library:", err)
		return
	}

	beego.Debug("Found", total, "tracks,",
		len(i.scanner.MediaFiles()), "songs,",
		len(i.scanner.Albums()), "albums,",
		len(i.scanner.Artists()), "artists",
		len(i.scanner.Playlists()), "playlists")

	if err := i.importLibrary(); err != nil {
		beego.Error("Error persisting data:", err)
	}
	beego.Info("Finished importing tracks from iTunes Library")
}

func (i *Importer) lastModifiedSince() time.Time {
	ms, err := i.propertyRepo.Get(consts.LastScan)
	if err != nil {
		beego.Warn("Couldn't read LastScan:", err)
		return time.Time{}
	}
	if ms == "" {
		beego.Debug("First scan")
		return time.Time{}
	}
	s, _ := strconv.ParseInt(ms, 10, 64)
	return time.Unix(0, s*int64(time.Millisecond))
}

func (i *Importer) importLibrary() (err error) {
	indexGroups := utils.ParseIndexGroups(beego.AppConfig.String("indexGroups"))
	artistIndex := make(map[string]tempIndex)
	mfs := make(domain.MediaFiles, len(i.scanner.MediaFiles()))
	als := make(domain.Albums, len(i.scanner.Albums()))
	ars := make(domain.Artists, len(i.scanner.Artists()))
	pls := make(domain.Playlists, len(i.scanner.Playlists()))

	i.search.ClearAll()

	beego.Debug("Saving updated data")
	j := 0
	for _, mf := range i.scanner.MediaFiles() {
		mfs[j] = *mf
		j++
		if mf.UpdatedAt.Before(i.lastScan) {
			continue
		}
		if err := i.mfRepo.Put(mf); err != nil {
			beego.Error(err)
		}
		if err := i.search.IndexMediaFile(mf); err != nil {
			beego.Error("Error indexing artist:", err)
		}
		if !i.lastScan.IsZero() {
			beego.Debug("Updated Track:", mf.Title)
		}
	}

	j = 0
	for _, al := range i.scanner.Albums() {
		als[j] = *al
		j++
		if al.UpdatedAt.Before(i.lastScan) {
			continue
		}
		if err := i.albumRepo.Put(al); err != nil {
			beego.Error(err)
		}
		if err := i.search.IndexAlbum(al); err != nil {
			beego.Error("Error indexing artist:", err)
		}
		if !i.lastScan.IsZero() {
			beego.Debug(fmt.Sprintf(`Updated Album:"%s" from "%s"`, al.Name, al.Artist))
		}
	}

	j = 0
	for _, ar := range i.scanner.Artists() {
		ars[j] = *ar
		j++
		if err := i.artistRepo.Put(ar); err != nil {
			beego.Error(err)
		}
		if err := i.search.IndexArtist(ar); err != nil {
			beego.Error("Error indexing artist:", err)
		}
		i.collectIndex(indexGroups, ar, artistIndex)
	}

	j = 0
	for _, pl := range i.scanner.Playlists() {
		pls[j] = *pl
		j++
		if err := i.plsRepo.Put(pl); err != nil {
			beego.Error(err)
		}
	}

	if err = i.saveIndex(artistIndex); err != nil {
		beego.Error(err)
	}

	beego.Debug("Purging old data")
	if err := i.mfRepo.PurgeInactive(&mfs); err != nil {
		beego.Error(err)
	}
	if err := i.albumRepo.PurgeInactive(&als); err != nil {
		beego.Error(err)
	}
	if err := i.artistRepo.PurgeInactive(&ars); err != nil {
		beego.Error(err)
	}
	if err := i.plsRepo.PurgeInactive(&pls); err != nil {
		beego.Error(err)
	}

	c, _ := i.artistRepo.CountAll()
	beego.Info("Total Artists in database:", c)
	c, _ = i.albumRepo.CountAll()
	beego.Info("Total Albums in database:", c)
	c, _ = i.mfRepo.CountAll()
	beego.Info("Total MediaFiles in database:", c)
	c, _ = i.plsRepo.CountAll()
	beego.Info("Total Playlists in database:", c)

	if err == nil {
		millis := time.Now().UnixNano() / int64(time.Millisecond)
		i.propertyRepo.Put(consts.LastScan, fmt.Sprint(millis))
		beego.Info("LastScan timestamp:", millis)
	}

	return err
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
