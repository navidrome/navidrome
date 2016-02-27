package scanner

import (
	"github.com/dhowden/itl"
	"net/url"
	"os"
	"strings"
)

type ItunesScanner struct {}

func (s *ItunesScanner) LoadFolder(path string) []Track {
	xml, _ := os.Open(path)
	l, _ := itl.ReadFromXML(xml)

	mediaFiles := make([]Track, len(l.Tracks))
	i := 0
	for id, t := range l.Tracks {
		if t.Location != "" && strings.Contains(t.Kind, "audio") {
			mediaFiles[i].Id = id
			mediaFiles[i].Album = t.Album
			mediaFiles[i].Title = t.Name
			mediaFiles[i].Artist = t.Artist
			mediaFiles[i].AlbumArtist = t.AlbumArtist
			mediaFiles[i].Compilation = t.Compilation
			path, _ = url.QueryUnescape(t.Location)
			mediaFiles[i].Path = strings.TrimPrefix(path, "file://")
			mediaFiles[i].CreatedAt = t.DateAdded
			mediaFiles[i].UpdatedAt = t.DateModified
			i++
		}
	}
	return mediaFiles[0:i]
}

var _ Scanner = (*ItunesScanner)(nil)