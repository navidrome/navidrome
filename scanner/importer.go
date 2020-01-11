package scanner

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cloudsonic/sonic-server/conf"
	"github.com/cloudsonic/sonic-server/domain"
	"github.com/cloudsonic/sonic-server/engine"
	"github.com/cloudsonic/sonic-server/log"
	"github.com/cloudsonic/sonic-server/utils"
)

type Scanner interface {
	ScanLibrary(lastModifiedSince time.Time, path string) (int, error)
	MediaFiles() map[string]*domain.MediaFile
	Albums() map[string]*domain.Album
	Artists() map[string]*domain.Artist
	Playlists() map[string]*domain.Playlist
}

type tempIndex map[string]domain.ArtistInfo

type Importer struct {
	scanner      Scanner
	mediaFolder  string
	mfRepo       domain.MediaFileRepository
	albumRepo    domain.AlbumRepository
	artistRepo   domain.ArtistRepository
	idxRepo      domain.ArtistIndexRepository
	plsRepo      domain.PlaylistRepository
	propertyRepo domain.PropertyRepository
	search       engine.Search
	lastScan     time.Time
	lastCheck    time.Time
}

func NewImporter(mediaFolder string, scanner Scanner, mfRepo domain.MediaFileRepository, albumRepo domain.AlbumRepository, artistRepo domain.ArtistRepository, idxRepo domain.ArtistIndexRepository, plsRepo domain.PlaylistRepository, propertyRepo domain.PropertyRepository, search engine.Search) *Importer {
	return &Importer{
		scanner:      scanner,
		mediaFolder:  mediaFolder,
		mfRepo:       mfRepo,
		albumRepo:    albumRepo,
		artistRepo:   artistRepo,
		idxRepo:      idxRepo,
		plsRepo:      plsRepo,
		propertyRepo: propertyRepo,
		search:       search,
	}
}

func (i *Importer) CheckForUpdates(force bool) {
	if force {
		i.lastCheck = time.Time{}
	}

	i.startImport()
}

func (i *Importer) startImport() {
	go func() {
		info, err := os.Stat(i.mediaFolder)
		if err != nil {
			log.Error(err)
			return
		}
		if i.lastCheck.After(info.ModTime()) {
			return
		}
		i.lastCheck = time.Now()

		i.scan()
	}()
}

func (i *Importer) scan() {
	i.lastScan = i.lastModifiedSince()

	if i.lastScan.IsZero() {
		log.Info("Starting first iTunes Library scan. This can take a while...")
	}

	total, err := i.scanner.ScanLibrary(i.lastScan, i.mediaFolder)
	if err != nil {
		log.Error("Error importing iTunes Library", err)
		return
	}

	log.Debug("Totals informed by the scanner", "tracks", total,
		"songs", len(i.scanner.MediaFiles()),
		"albums", len(i.scanner.Albums()),
		"artists", len(i.scanner.Artists()),
		"playlists", len(i.scanner.Playlists()))

	if err := i.importLibrary(); err != nil {
		log.Error("Error persisting data", err)
	}
	if i.lastScan.IsZero() {
		log.Info("Finished first iTunes Library import")
	} else {
		log.Debug("Finished updating tracks from iTunes Library")
	}
}

func (i *Importer) lastModifiedSince() time.Time {
	ms, err := i.propertyRepo.Get(domain.PropLastScan)
	if err != nil {
		log.Warn("Couldn't read LastScan", err)
		return time.Time{}
	}
	if ms == "" {
		log.Debug("First scan")
		return time.Time{}
	}
	s, _ := strconv.ParseInt(ms, 10, 64)
	return time.Unix(0, s*int64(time.Millisecond))
}

func (i *Importer) importLibrary() (err error) {
	arc, _ := i.artistRepo.CountAll()
	alc, _ := i.albumRepo.CountAll()
	mfc, _ := i.mfRepo.CountAll()
	plc, _ := i.plsRepo.CountAll()

	log.Debug("Saving updated data")
	mfs, mfu := i.importMediaFiles()
	log.Debug("Imported media files", "total", len(mfs), "updated", mfu)
	als, alu := i.importAlbums()
	log.Debug("Imported albums", "total", len(als), "updated", alu)
	ars := i.importArtists()
	log.Debug("Imported artists", "total", len(ars))
	pls := i.importPlaylists()
	log.Debug("Imported playlists", "total", len(pls))
	i.importArtistIndex()

	log.Debug("Purging old data")
	if deleted, err := i.mfRepo.PurgeInactive(mfs); err != nil {
		log.Error(err)
	} else {
		i.search.RemoveMediaFile(deleted...)
	}
	if deleted, err := i.albumRepo.PurgeInactive(als); err != nil {
		log.Error(err)
	} else {
		i.search.RemoveAlbum(deleted...)
	}
	if deleted, err := i.artistRepo.PurgeInactive(ars); err != nil {
		log.Error("Deleting inactive artists", err)
	} else {
		i.search.RemoveArtist(deleted...)
	}
	if _, err := i.plsRepo.PurgeInactive(pls); err != nil {
		log.Error(err)
	}

	arc2, _ := i.artistRepo.CountAll()
	alc2, _ := i.albumRepo.CountAll()
	mfc2, _ := i.mfRepo.CountAll()
	plc2, _ := i.plsRepo.CountAll()

	if arc != arc2 || alc != alc2 || mfc != mfc2 || plc != plc2 {
		log.Info(fmt.Sprintf("Updated library totals: %d(%+d) artists, %d(%+d) albums, %d(%+d) songs, %d(%+d) playlists", arc2, arc2-arc, alc2, alc2-alc, mfc2, mfc2-mfc, plc2, plc2-plc))
	}
	if alu > 0 || mfu > 0 {
		log.Info(fmt.Sprintf("Updated items: %d album(s), %d song(s)", alu, mfu))
	}

	if err == nil {
		millis := time.Now().UnixNano() / int64(time.Millisecond)
		i.propertyRepo.Put(domain.PropLastScan, fmt.Sprint(millis))
		log.Debug("LastScan", "timestamp", millis)
	}

	return err
}

func (i *Importer) importMediaFiles() (domain.MediaFiles, int) {
	mfs := make(domain.MediaFiles, len(i.scanner.MediaFiles()))
	updates := 0
	j := 0
	for _, mf := range i.scanner.MediaFiles() {
		mfs[j] = *mf
		j++
		if mf.UpdatedAt.Before(i.lastScan) {
			continue
		}
		if mf.Starred {
			original, err := i.mfRepo.Get(mf.ID)
			if err != nil || !original.Starred {
				mf.StarredAt = mf.UpdatedAt
			} else {
				mf.StarredAt = original.StarredAt
			}
		}
		if err := i.mfRepo.Put(mf); err != nil {
			log.Error(err)
		}
		if err := i.search.IndexMediaFile(mf); err != nil {
			log.Error("Error indexing artist", err)
		}
		updates++
		if !i.lastScan.IsZero() {
			log.Debug(fmt.Sprintf(`-- Updated Track: "%s"`, mf.Title))
		}
	}
	return mfs, updates
}

func (i *Importer) importAlbums() (domain.Albums, int) {
	als := make(domain.Albums, len(i.scanner.Albums()))
	updates := 0
	j := 0
	for _, al := range i.scanner.Albums() {
		als[j] = *al
		j++
		if al.UpdatedAt.Before(i.lastScan) {
			continue
		}
		if al.Starred {
			original, err := i.albumRepo.Get(al.ID)
			if err != nil || !original.Starred {
				al.StarredAt = al.UpdatedAt
			} else {
				al.StarredAt = original.StarredAt
			}
		}
		if err := i.albumRepo.Put(al); err != nil {
			log.Error(err)
		}
		if err := i.search.IndexAlbum(al); err != nil {
			log.Error("Error indexing artist", err)
		}
		updates++
		if !i.lastScan.IsZero() {
			log.Debug(fmt.Sprintf(`-- Updated Album: "%s" from "%s"`, al.Name, al.Artist))
		}
	}
	return als, updates
}

func (i *Importer) importArtists() domain.Artists {
	ars := make(domain.Artists, len(i.scanner.Artists()))
	j := 0
	for _, ar := range i.scanner.Artists() {
		ars[j] = *ar
		j++
		if err := i.artistRepo.Put(ar); err != nil {
			log.Error(err)
		}
		if err := i.search.IndexArtist(ar); err != nil {
			log.Error("Error indexing artist", err)
		}
	}
	return ars
}

func (i *Importer) importArtistIndex() {
	indexGroups := utils.ParseIndexGroups(conf.Sonic.IndexGroups)
	artistIndex := make(map[string]tempIndex)

	for _, ar := range i.scanner.Artists() {
		i.collectIndex(indexGroups, ar, artistIndex)
	}

	if err := i.saveIndex(artistIndex); err != nil {
		log.Error(err)
	}
}

func (i *Importer) importPlaylists() domain.Playlists {
	pls := make(domain.Playlists, len(i.scanner.Playlists()))
	j := 0
	for _, pl := range i.scanner.Playlists() {
		pl.Public = true
		pl.Owner = conf.Sonic.User
		pl.Comment = "Original: " + pl.FullPath
		pls[j] = *pl
		j++
		if err := i.plsRepo.Put(pl); err != nil {
			log.Error(err)
		}
	}
	return pls
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
	artists[indexName] = domain.ArtistInfo{ArtistID: a.ID, Artist: a.Name, AlbumCount: a.AlbumCount}
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
	i.idxRepo.DeleteAll()
	for k, temp := range artistIndex {
		idx := &domain.ArtistIndex{ID: k}
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
