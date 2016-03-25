package scanner

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/astaxie/beego"
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

var (
	inProgress    chan int
	lastCheck     time.Time
	itunesLibrary string
)

func init() {
	inProgress = make(chan int)
	go func() {
		time.Sleep(5 * time.Second)
		startImport()
	}()
}

func CheckForUpdates(force bool) {
	<-inProgress

	if force {
		lastCheck = time.Time{}
	}

	startImport()
}

func startImport() {
	go func() {
		itunesLibrary = beego.AppConfig.String("musicFolder")

		info, err := os.Stat(itunesLibrary)
		if err != nil {
			inProgress <- 1
			beego.Error(err)
			return
		}
		if lastCheck.After(info.ModTime()) {
			inProgress <- 1
			return
		}
		lastCheck = time.Now()

		// TODO Move all to DI
		i := &Importer{mediaFolder: beego.AppConfig.String("musicFolder")}
		utils.ResolveDependencies(&i.mfRepo, &i.albumRepo, &i.artistRepo, &i.idxRepo, &i.plsRepo,
			&i.propertyRepo, &i.search, &i.scanner)
		i.Run()
		inProgress <- 1
	}()
}

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

	if i.lastScan.IsZero() {
		beego.Info("Starting first iTunes Library scan. This can take a while...")
	}

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
	if i.lastScan.IsZero() {
		beego.Info("Finished first iTunes Library import")
	} else {
		beego.Debug("Finished updating tracks from iTunes Library")
	}
}

func (i *Importer) lastModifiedSince() time.Time {
	ms, err := i.propertyRepo.Get(engine.PropLastScan)
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
	arc, _ := i.artistRepo.CountAll()
	alc, _ := i.albumRepo.CountAll()
	mfc, _ := i.mfRepo.CountAll()
	plc, _ := i.plsRepo.CountAll()
	alu := 0
	mfu := 0

	//i.search.ClearAll()
	//
	beego.Debug("Saving updated data")
	j := 0
	for _, mf := range i.scanner.MediaFiles() {
		mfs[j] = *mf
		j++
		if mf.UpdatedAt.Before(i.lastScan) {
			continue
		}
		if mf.Starred {
			original, err := i.mfRepo.Get(mf.Id)
			if err != nil || !original.Starred {
				mf.StarredAt = mf.UpdatedAt
			} else {
				mf.StarredAt = original.StarredAt
			}
		}
		if err := i.mfRepo.Put(mf); err != nil {
			beego.Error(err)
		}
		if err := i.search.IndexMediaFile(mf); err != nil {
			beego.Error("Error indexing artist:", err)
		}
		mfu++
		if !i.lastScan.IsZero() {
			beego.Debug(fmt.Sprintf(`-- Updated Track: "%s"`, mf.Title))
		}
	}

	j = 0
	for _, al := range i.scanner.Albums() {
		als[j] = *al
		j++
		if al.UpdatedAt.Before(i.lastScan) {
			continue
		}
		if al.Starred {
			original, err := i.albumRepo.Get(al.Id)
			if err != nil || !original.Starred {
				al.StarredAt = al.UpdatedAt
			} else {
				al.StarredAt = original.StarredAt
			}
		}
		if err := i.albumRepo.Put(al); err != nil {
			beego.Error(err)
		}
		if err := i.search.IndexAlbum(al); err != nil {
			beego.Error("Error indexing artist:", err)
		}
		alu++
		if !i.lastScan.IsZero() {
			beego.Debug(fmt.Sprintf(`-- Updated Album:"%s" from "%s"`, al.Name, al.Artist))
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
		pl.Public = true
		pl.Owner = beego.AppConfig.String("user")
		pl.Comment = "Original: " + pl.FullPath
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
	if deleted, err := i.mfRepo.PurgeInactive(mfs); err != nil {
		beego.Error(err)
	} else {
		i.search.RemoveMediaFile(deleted)
	}
	if deleted, err := i.albumRepo.PurgeInactive(als); err != nil {
		beego.Error(err)
	} else {
		i.search.RemoveAlbum(deleted)
	}
	if deleted, err := i.artistRepo.PurgeInactive(ars); err != nil {
		beego.Error(err)
	} else {
		i.search.RemoveArtist(deleted)
	}
	if _, err := i.plsRepo.PurgeInactive(pls); err != nil {
		beego.Error(err)
	}

	arc2, _ := i.artistRepo.CountAll()
	alc2, _ := i.albumRepo.CountAll()
	mfc2, _ := i.mfRepo.CountAll()
	plc2, _ := i.plsRepo.CountAll()

	if arc != arc2 || alc != alc2 || mfc != mfc2 || plc != plc2 {
		beego.Info(fmt.Sprintf("Updated library totals: %d(%+d) artists, %d(%+d) albums, %d(%+d) songs, %d(%+d) playlists", arc2, arc2-arc, alc2, alc2-alc, mfc2, mfc2-mfc, plc2, plc2-plc))
	}
	if alu > 0 || mfu > 0 {
		beego.Info(fmt.Sprintf("Updated items: %d album(s), %d song(s)", alu, mfu))
	}

	if err == nil {
		millis := time.Now().UnixNano() / int64(time.Millisecond)
		i.propertyRepo.Put(engine.PropLastScan, fmt.Sprint(millis))
		beego.Debug("LastScan timestamp:", millis)
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
