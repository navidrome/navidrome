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
			mediaFiles[i].Album = unescape(t.Album)
			mediaFiles[i].Title = unescape(t.Name)
			mediaFiles[i].Artist = unescape(t.Artist)
			mediaFiles[i].AlbumArtist = unescape(t.AlbumArtist)
			mediaFiles[i].Compilation = t.Compilation
			mediaFiles[i].Year = t.Year
			path, _ = url.QueryUnescape(t.Location)
			mediaFiles[i].Path = strings.TrimPrefix(path, "file://")
			mediaFiles[i].CreatedAt = t.DateAdded
			mediaFiles[i].UpdatedAt = t.DateModified
			i++
		}
	}
	return mediaFiles[0:i]
}

func unescape(s string) string {
	s,_ = url.QueryUnescape(s)
	return strings.Replace(s, "&#38;", "&", -1)
}

var _ Scanner = (*ItunesScanner)(nil)