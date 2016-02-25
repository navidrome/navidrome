package itunes

import (
	"github.com/deluan/gosonic/models"
	"github.com/dhowden/itl"
	"net/url"
	"os"
	"strings"
)

func LoadFolder(path string) []models.MediaFile {
	xml, _ := os.Open(path)
	l, _ := itl.ReadFromXML(xml)

	mediaFiles := make([]models.MediaFile, len(l.Tracks))
	i := 0
	for id, track := range l.Tracks {
		mediaFiles[i].Id = id
		mediaFiles[i].Album = track.Album
		mediaFiles[i].Title = track.Name
		mediaFiles[i].Artist = track.Artist
		path, _ = url.QueryUnescape(track.Location)
		mediaFiles[i].Path = strings.TrimPrefix(path, "file://")
		mediaFiles[i].CreatedAt = track.DateAdded
		mediaFiles[i].UpdatedAt = track.DateModified
		i++
	}
	return mediaFiles
}
