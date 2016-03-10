package scanner

import (
	"crypto/md5"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"html"

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
	lastModifiedSince time.Time
}

func (s *ItunesScanner) ScanLibrary(lastModifiedSince time.Time, path string) (int, error) {
	beego.Info("Starting iTunes import from:", path)
	beego.Info("Checking for updates since", lastModifiedSince.String())
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

	i := 0
	for _, t := range l.Tracks {
		if strings.HasPrefix(t.Location, "file://") && strings.Contains(t.Kind, "audio") {
			ar := s.collectArtists(&t)
			mf := s.collectMediaFiles(&t)
			s.collectAlbums(&t, mf, ar)
		}
		i++
		if i%1000 == 0 {
			beego.Debug("Processed", i, "tracks.", len(s.artists), "artists,", len(s.albums), "albums", len(s.mediaFiles), "songs")
		}
	}

	ignFolders, _ := beego.AppConfig.Bool("ignorePlsFolders")
	for _, p := range l.Playlists {
		if p.Master || p.Music || (ignFolders && p.Folder) {
			continue
		}

		s.collectPlaylists(&p)
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

func (s *ItunesScanner) collectPlaylists(p *itl.Playlist) {
	pl := &domain.Playlist{}
	pl.Id = strconv.Itoa(p.PlaylistID)
	pl.Name = p.Name
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

func (s *ItunesScanner) collectMediaFiles(t *itl.Track) *domain.MediaFile {
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

	path, _ := url.QueryUnescape(t.Location)
	path = strings.TrimPrefix(unescape(path), "file://")
	mf.Path = path
	mf.Suffix = strings.TrimPrefix(filepath.Ext(path), ".")

	mf.CreatedAt = t.DateAdded
	mf.UpdatedAt = t.DateModified

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
	if al.UpdatedAt.IsZero() || t.DateModified.After(al.UpdatedAt) {
		al.UpdatedAt = t.DateModified
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
