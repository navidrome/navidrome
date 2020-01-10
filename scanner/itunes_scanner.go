package scanner

import (
	"crypto/md5"
	"fmt"
	"html"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cloudsonic/sonic-server/conf"
	"github.com/cloudsonic/sonic-server/domain"
	"github.com/cloudsonic/sonic-server/log"
	"github.com/dhowden/itl"
	"github.com/dhowden/tag"
)

type ItunesScanner struct {
	mediaFiles        map[string]*domain.MediaFile
	albums            map[string]*domain.Album
	artists           map[string]*domain.Artist
	playlists         map[string]*domain.Playlist
	pplaylists        map[string]plsRelation
	pmediaFiles       map[int]*domain.MediaFile
	lastModifiedSince time.Time
	checksumRepo      CheckSumRepository
	newSums           map[string]string
}

func NewItunesScanner(checksumRepo CheckSumRepository) *ItunesScanner {
	return &ItunesScanner{checksumRepo: checksumRepo}
}

type CheckSumRepository interface {
	Put(id, sum string) error
	Get(id string) (string, error)
	SetData(newSums map[string]string) error
}

type plsRelation struct {
	pID       string
	parentPID string
	name      string
}

func (s *ItunesScanner) ScanLibrary(lastModifiedSince time.Time, path string) (int, error) {
	log.Debug("Checking for updates", "lastModifiedSince", lastModifiedSince, "library", path)
	xml, _ := os.Open(path)
	l, err := itl.ReadFromXML(xml)
	if err != nil {
		return 0, err
	}
	log.Debug("Loaded tracks", "total", len(l.Tracks))

	s.lastModifiedSince = lastModifiedSince
	s.mediaFiles = make(map[string]*domain.MediaFile)
	s.albums = make(map[string]*domain.Album)
	s.artists = make(map[string]*domain.Artist)
	s.playlists = make(map[string]*domain.Playlist)
	s.pplaylists = make(map[string]plsRelation)
	s.pmediaFiles = make(map[int]*domain.MediaFile)
	s.newSums = make(map[string]string)
	songsPerAlbum := make(map[string]int)
	albumsPerArtist := make(map[string]map[string]bool)

	i := 0
	for _, t := range l.Tracks {
		if !s.skipTrack(&t) {
			s.calcCheckSum(&t)

			ar := s.collectArtists(&t)
			mf := s.collectMediaFiles(&t)
			s.collectAlbums(&t, mf, ar)

			songsPerAlbum[mf.AlbumID]++
			if albumsPerArtist[mf.ArtistID] == nil {
				albumsPerArtist[mf.ArtistID] = make(map[string]bool)
			}
			albumsPerArtist[mf.ArtistID][mf.AlbumID] = true
		}
		i++
		if i%1000 == 0 {
			log.Debug(fmt.Sprintf("Processed %d tracks", i), "artists", len(s.artists), "albums", len(s.albums), "songs", len(s.mediaFiles))
		}
	}

	for albumId, count := range songsPerAlbum {
		s.albums[albumId].SongCount = count
	}

	for artistId, albums := range albumsPerArtist {
		s.artists[artistId].AlbumCount = len(albums)
	}

	if err := s.checksumRepo.SetData(s.newSums); err != nil {
		log.Error("Error saving checksums", err)
	} else {
		log.Debug("Saved checksums", "total", len(s.newSums))
	}

	ignFolders := conf.Sonic.PlsIgnoreFolders
	ignPatterns := strings.Split(conf.Sonic.PlsIgnoredPatterns, ";")
	for _, p := range l.Playlists {
		rel := plsRelation{pID: p.PlaylistPersistentID, parentPID: p.ParentPersistentID, name: unescape(p.Name)}
		s.pplaylists[p.PlaylistPersistentID] = rel
		fullPath := s.fullPath(p.PlaylistPersistentID)

		if s.skipPlaylist(&p, ignFolders, ignPatterns, fullPath) {
			continue
		}

		s.collectPlaylists(&p, fullPath)
	}
	log.Debug("Processed playlists", "total", len(l.Playlists))

	return len(l.Tracks), nil
}

func (s *ItunesScanner) MediaFiles() map[string]*domain.MediaFile {
	return s.mediaFiles
}
func (s *ItunesScanner) Albums() map[string]*domain.Album {
	return s.albums
}
func (s *ItunesScanner) Artists() map[string]*domain.Artist {
	return s.artists
}
func (s *ItunesScanner) Playlists() map[string]*domain.Playlist {
	return s.playlists
}

func (s *ItunesScanner) skipTrack(t *itl.Track) bool {
	if t.Podcast {
		return true
	}

	if conf.Sonic.DevDisableFileCheck {
		return false
	}

	if !strings.HasPrefix(t.Location, "file://") {
		return true
	}

	ext := filepath.Ext(t.Location)
	m := mime.TypeByExtension(ext)

	return !strings.HasPrefix(m, "audio/")
}

func (s *ItunesScanner) skipPlaylist(p *itl.Playlist, ignFolders bool, ignPatterns []string, fullPath string) bool {
	// Skip all "special" iTunes playlists, and also ignored patterns
	if p.Master || p.Music || p.Audiobooks || p.Movies || p.TVShows || p.Podcasts || p.ITunesU || (ignFolders && p.Folder) {
		return true
	}

	for _, p := range ignPatterns {
		if p == "" {
			continue
		}
		m, _ := regexp.MatchString(p, fullPath)
		if m {
			return true
		}
	}

	return false
}

func (s *ItunesScanner) collectPlaylists(p *itl.Playlist, fullPath string) {
	pl := &domain.Playlist{}
	pl.ID = p.PlaylistPersistentID
	pl.Name = unescape(p.Name)
	pl.FullPath = fullPath
	pl.Tracks = make([]string, 0, len(p.PlaylistItems))
	for _, item := range p.PlaylistItems {
		if mf, found := s.pmediaFiles[item.TrackID]; found {
			pl.Tracks = append(pl.Tracks, mf.ID)
			pl.Duration += mf.Duration
		}
	}
	if len(pl.Tracks) > 0 {
		s.playlists[pl.ID] = pl
	}
}

func (s *ItunesScanner) fullPath(pID string) string {
	rel, found := s.pplaylists[pID]
	if !found {
		return ""
	}
	if rel.parentPID == "" {
		return rel.name
	}
	return fmt.Sprintf("%s > %s", s.fullPath(rel.parentPID), rel.name)
}

func (s *ItunesScanner) lastChangedDate(t *itl.Track) time.Time {
	if s.hasChanged(t) {
		return time.Now()
	}
	allDates := []time.Time{t.DateModified, t.PlayDateUTC}
	c := time.Time{}
	for _, d := range allDates {
		if c.Before(d) {
			c = d
		}
	}
	return c
}

func (s *ItunesScanner) hasChanged(t *itl.Track) bool {
	id := t.PersistentID
	oldSum, _ := s.checksumRepo.Get(id)
	newSum := s.newSums[id]
	return oldSum != newSum
}

// Calc sum of stats fields (whose changes are not reflected in DataModified)
func (s *ItunesScanner) calcCheckSum(t *itl.Track) string {
	id := t.PersistentID
	data := fmt.Sprint(t.DateModified, t.PlayCount, t.PlayDate, t.ArtworkCount, t.Loved, t.AlbumLoved,
		t.Rating, t.AlbumRating, t.SkipCount, t.SkipDate)
	sum := fmt.Sprintf("%x", md5.Sum([]byte(data)))
	s.newSums[id] = sum
	return sum
}

func (s *ItunesScanner) collectMediaFiles(t *itl.Track) *domain.MediaFile {
	mf := &domain.MediaFile{}
	mf.ID = t.PersistentID
	mf.Album = unescape(t.Album)
	mf.AlbumID = albumId(t)
	mf.ArtistID = artistId(t)
	mf.Title = unescape(t.Name)
	mf.Artist = unescape(t.Artist)
	if mf.Album == "" {
		mf.Album = "[Unknown Album]"
	}
	if mf.Artist == "" {
		mf.Artist = "[Unknown Artist]"
	}
	mf.AlbumArtist = unescape(t.AlbumArtist)
	mf.Genre = unescape(t.Genre)
	mf.Compilation = t.Compilation
	mf.Starred = t.Loved
	mf.Rating = t.Rating / 20
	mf.PlayCount = t.PlayCount
	mf.PlayDate = t.PlayDateUTC
	mf.Year = t.Year
	mf.TrackNumber = t.TrackNumber
	mf.DiscNumber = t.DiscNumber
	if t.Size > 0 {
		mf.Size = strconv.Itoa(t.Size)
	}
	if t.TotalTime > 0 {
		mf.Duration = t.TotalTime / 1000
	}
	mf.BitRate = t.BitRate

	path := extractPath(t.Location)
	mf.Path = path
	mf.Suffix = strings.TrimPrefix(filepath.Ext(path), ".")

	mf.CreatedAt = t.DateAdded
	mf.UpdatedAt = s.lastChangedDate(t)

	if mf.UpdatedAt.After(s.lastModifiedSince) && !conf.Sonic.DevDisableFileCheck {
		mf.HasCoverArt = hasCoverArt(path)
	}

	s.mediaFiles[mf.ID] = mf
	s.pmediaFiles[t.TrackID] = mf

	return mf
}

func (s *ItunesScanner) collectAlbums(t *itl.Track, mf *domain.MediaFile, ar *domain.Artist) *domain.Album {
	id := albumId(t)
	_, found := s.albums[id]
	if !found {
		s.albums[id] = &domain.Album{}
	}

	al := s.albums[id]
	al.ID = id
	al.ArtistID = ar.ID
	al.Name = mf.Album
	al.Year = t.Year
	al.Compilation = t.Compilation
	al.Starred = t.AlbumLoved
	al.Rating = t.AlbumRating / 20
	al.PlayCount += t.PlayCount
	al.Genre = mf.Genre
	al.Artist = mf.Artist
	al.AlbumArtist = ar.Name
	if al.Name == "" {
		al.Name = "[Unknown Album]"
	}
	if al.Artist == "" {
		al.Artist = "[Unknown Artist]"
	}
	al.Duration += mf.Duration

	if mf.HasCoverArt {
		al.CoverArtId = mf.ID
		al.CoverArtPath = mf.Path
	}

	if al.PlayDate.IsZero() || t.PlayDateUTC.After(al.PlayDate) {
		al.PlayDate = t.PlayDateUTC
	}
	if al.CreatedAt.IsZero() || t.DateAdded.Before(al.CreatedAt) {
		al.CreatedAt = t.DateAdded
	}
	trackUpdate := s.lastChangedDate(t)
	if al.UpdatedAt.IsZero() || trackUpdate.After(al.UpdatedAt) {
		al.UpdatedAt = trackUpdate
	}

	return al
}

func (s *ItunesScanner) collectArtists(t *itl.Track) *domain.Artist {
	id := artistId(t)
	_, found := s.artists[id]
	if !found {
		s.artists[id] = &domain.Artist{}
	}
	ar := s.artists[id]
	ar.ID = id
	ar.Name = unescape(realArtistName(t))
	if ar.Name == "" {
		ar.Name = "[Unknown Artist]"
	}

	return ar
}

func albumId(t *itl.Track) string {
	s := strings.ToLower(fmt.Sprintf("%s\\%s", realArtistName(t), t.Album))
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

func artistId(t *itl.Track) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(strings.ToLower(realArtistName(t)))))
}

func hasCoverArt(path string) bool {
	defer func() {
		if r := recover(); r != nil {
			log.Error("Panic reading tag", "path", path, "error", r)
		}
	}()

	if _, err := os.Stat(path); err == nil {
		f, err := os.Open(path)
		if err != nil {
			log.Warn("Error opening file", "path", path, err)
			return false
		}
		defer f.Close()

		m, err := tag.ReadFrom(f)
		if err != nil {
			log.Warn("Error reading tag from file", "path", path, err)
			return false
		}
		return m.Picture() != nil
	}
	//log.Warn("File not found", "path", path)
	return false
}

func unescape(str string) string {
	return html.UnescapeString(str)
}

func extractPath(loc string) string {
	path := strings.Replace(loc, "+", "%2B", -1)
	path, _ = url.QueryUnescape(path)
	path = html.UnescapeString(path)
	return strings.TrimPrefix(path, "file://")
}

func realArtistName(t *itl.Track) string {
	switch {
	case t.Compilation:
		return "Various Artists"
	case t.AlbumArtist != "":
		return t.AlbumArtist
	}

	return t.Artist
}

var _ Scanner = (*ItunesScanner)(nil)
