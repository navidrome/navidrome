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

	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/itl"
	"github.com/dhowden/tag"
)

type ItunesScanner struct {
	mediaFiles        map[string]*domain.MediaFile
	albums            map[string]*domain.Album
	artists           map[string]*domain.Artist
	playlists         map[string]*domain.Playlist
	pplaylists        map[string]plsRelation
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
	beego.Info("Checking for updates since", lastModifiedSince.String(), "- Library:", path)
	xml, _ := os.Open(path)
	l, err := itl.ReadFromXML(xml)
	if err != nil {
		return 0, err
	}
	beego.Debug("Loaded", len(l.Tracks), "tracks")

	s.lastModifiedSince = lastModifiedSince
	s.mediaFiles = make(map[string]*domain.MediaFile)
	s.albums = make(map[string]*domain.Album)
	s.artists = make(map[string]*domain.Artist)
	s.playlists = make(map[string]*domain.Playlist)
	s.pplaylists = make(map[string]plsRelation)
	s.newSums = make(map[string]string)

	i := 0
	for _, t := range l.Tracks {
		if !s.skipTrack(&t) {
			ar := s.collectArtists(&t)
			mf := s.collectMediaFiles(&t)
			s.collectAlbums(&t, mf, ar)
		}
		i++
		if i%1000 == 0 {
			beego.Debug("Processed", i, "tracks.", len(s.artists), "artists,", len(s.albums), "albums", len(s.mediaFiles), "songs")
		}
	}

	if err := s.checksumRepo.SetData(s.newSums); err != nil {
		beego.Error("Error saving checksums:", err)
	} else {
		beego.Debug("Saved", len(s.newSums), "checksums")
	}

	ignFolders, _ := beego.AppConfig.Bool("plsIgnoreFolders")
	ignPatterns := beego.AppConfig.Strings("plsIgnoredPatterns")
	for _, p := range l.Playlists {
		rel := plsRelation{pID: p.PlaylistPersistentID, parentPID: p.ParentPersistentID, name: unescape(p.Name)}
		s.pplaylists[p.PlaylistPersistentID] = rel
		fullPath := s.fullPath(p.PlaylistPersistentID)

		if s.skipPlaylist(&p, ignFolders, ignPatterns, fullPath) {
			continue
		}

		s.collectPlaylists(&p, fullPath)
	}
	beego.Debug("Processed", len(l.Playlists), "playlists.")

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
	if !strings.HasPrefix(t.Location, "file://") || t.Podcast {
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
		m, _ := regexp.MatchString(p, fullPath)
		if m {
			return true
		}
	}

	return false
}

func (s *ItunesScanner) collectPlaylists(p *itl.Playlist, fullPath string) {
	pl := &domain.Playlist{}
	pl.Id = strconv.Itoa(p.PlaylistID)
	pl.Name = unescape(p.Name)
	pl.FullPath = fullPath
	pl.Tracks = make([]string, 0, len(p.PlaylistItems))
	for _, item := range p.PlaylistItems {
		id := strconv.Itoa(item.TrackID)
		if _, found := s.mediaFiles[id]; found {
			pl.Tracks = append(pl.Tracks, id)
		}
	}
	if len(pl.Tracks) > 0 {
		s.playlists[pl.Id] = pl
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
	id := strconv.Itoa(t.TrackID)
	oldSum, _ := s.checksumRepo.Get(id)
	newSum := s.newSums[id]
	return oldSum != newSum
}

// Calc sum of stats fields (whose changes are not reflected in DataModified)
func (s *ItunesScanner) calcCheckSum(t *itl.Track) string {
	id := strconv.Itoa(t.TrackID)
	data := fmt.Sprint(t.DateModified, t.PlayCount, t.PlayDate, t.ArtworkCount, t.Loved, t.AlbumLoved,
		t.Rating, t.AlbumRating, t.SkipCount, t.SkipDate)
	sum := fmt.Sprintf("%x", md5.Sum([]byte(data)))
	s.newSums[id] = sum
	return sum
}

func (s *ItunesScanner) collectMediaFiles(t *itl.Track) *domain.MediaFile {
	s.calcCheckSum(t)

	mf := &domain.MediaFile{}
	mf.Id = strconv.Itoa(t.TrackID)
	mf.Album = unescape(t.Album)
	mf.AlbumId = albumId(t)
	mf.Title = unescape(t.Name)
	mf.Artist = unescape(t.Artist)
	mf.AlbumArtist = unescape(t.AlbumArtist)
	mf.Genre = unescape(t.Genre)
	mf.Compilation = t.Compilation
	mf.Starred = t.Loved
	mf.Rating = t.Rating
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

	if mf.UpdatedAt.After(s.lastModifiedSince) {
		mf.HasCoverArt = hasCoverArt(path)
	}

	s.mediaFiles[mf.Id] = mf

	return mf
}

func (s *ItunesScanner) collectAlbums(t *itl.Track, mf *domain.MediaFile, ar *domain.Artist) *domain.Album {
	id := albumId(t)
	_, found := s.albums[id]
	if !found {
		s.albums[id] = &domain.Album{}
	}

	al := s.albums[id]
	al.Id = id
	al.ArtistId = ar.Id
	al.Name = mf.Album
	al.Year = t.Year
	al.Compilation = t.Compilation
	al.Starred = t.AlbumLoved
	al.Rating = t.AlbumRating
	al.PlayCount += t.PlayCount
	al.Genre = mf.Genre
	al.Artist = mf.Artist
	al.AlbumArtist = ar.Name

	if mf.HasCoverArt {
		al.CoverArtId = mf.Id
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
	ar.Id = id
	ar.Name = unescape(realArtistName(t))

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
			beego.Error("Reading tag for", path, "Panic:", r)
		}
	}()

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
			return false
		}
		return m.Picture() != nil
	}
	//beego.Warn("File not found:", path)
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
