package scanner

import (
	"github.com/deluan/itl"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type ItunesScanner struct{}

func (s *ItunesScanner) LoadFolder(path string) []Track {
	xml, _ := os.Open(path)
	l, _ := itl.ReadFromXML(xml)

	mediaFiles := make([]Track, len(l.Tracks))
	i := 0
	for id, t := range l.Tracks {
		if strings.HasPrefix(t.Location, "file://") && strings.Contains(t.Kind, "audio") {
			mediaFiles[i].Id = id
			mediaFiles[i].Album = unescape(t.Album)
			mediaFiles[i].Title = unescape(t.Name)
			mediaFiles[i].Artist = unescape(t.Artist)
			mediaFiles[i].AlbumArtist = unescape(t.AlbumArtist)
			mediaFiles[i].Genre = unescape(t.Genre)
			mediaFiles[i].Compilation = t.Compilation
			mediaFiles[i].Loved = t.Loved
			mediaFiles[i].AlbumLoved = t.AlbumLoved
			mediaFiles[i].Year = t.Year
			mediaFiles[i].TrackNumber = t.TrackNumber
			mediaFiles[i].DiscNumber = t.DiscNumber
			if t.Size > 0 {
				mediaFiles[i].Size = strconv.Itoa(t.Size)
			}
			if t.TotalTime > 0 {
				mediaFiles[i].Duration = t.TotalTime / 1000
			}
			mediaFiles[i].BitRate = t.BitRate
			path, _ = url.QueryUnescape(t.Location)
			path = strings.TrimPrefix(unescape(path), "file://")
			mediaFiles[i].Path = path
			mediaFiles[i].Suffix = strings.TrimPrefix(filepath.Ext(path), ".")
			mediaFiles[i].CreatedAt = t.DateAdded
			mediaFiles[i].UpdatedAt = t.DateModified
			i++
		}
	}
	return mediaFiles[0:i]
}

func unescape(str string) string {
	s := strings.Replace(str, "&#38;", "&", -1)
	return s
}

var _ Scanner = (*ItunesScanner)(nil)
