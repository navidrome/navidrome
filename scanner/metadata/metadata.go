package metadata

import (
	"fmt"
	"math"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/scanner/metadata/ffmpeg"
	"github.com/navidrome/navidrome/scanner/metadata/taglib"
)

type Parser interface {
	Parse(files ...string) (map[string]map[string][]string, error)
}

var parsers = map[string]Parser{
	"ffmpeg": &ffmpeg.Parser{},
	"taglib": &taglib.Parser{},
}

func Extract(files ...string) (map[string]Tags, error) {
	p, ok := parsers[conf.Server.Scanner.Extractor]
	if !ok {
		log.Warn("Invalid 'Scanner.Extractor' option. Using default", "requested", conf.Server.Scanner.Extractor,
			"validOptions", "ffmpeg,taglib", "default", consts.DefaultScannerExtractor)
		p = parsers[consts.DefaultScannerExtractor]
	}

	extractedTags, err := p.Parse(files...)
	if err != nil {
		return nil, err
	}

	result := map[string]Tags{}
	for filePath, tags := range extractedTags {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			log.Warn("Error stating file. Skipping", "filePath", filePath, err)
			continue
		}

		result[filePath] = Tags{
			filePath: filePath,
			fileInfo: fileInfo,
			tags:     tags,
		}
	}

	return result, nil
}

type Tags struct {
	filePath string
	fileInfo os.FileInfo
	tags     map[string][]string
}

// Common tags

func (t Tags) Title() string  { return t.getFirstTagValue("title", "sort_name", "titlesort") }
func (t Tags) Album() string  { return t.getFirstTagValue("album", "sort_album", "albumsort") }
func (t Tags) Artist() string { return t.getFirstTagValue("artist", "sort_artist", "artistsort") }
func (t Tags) AlbumArtist() string {
	return t.getFirstTagValue("album_artist", "album artist", "albumartist")
}
func (t Tags) SortTitle() string       { return t.getSortTag("", "title", "name") }
func (t Tags) SortAlbum() string       { return t.getSortTag("", "album") }
func (t Tags) SortArtist() string      { return t.getSortTag("", "artist") }
func (t Tags) SortAlbumArtist() string { return t.getSortTag("tso2", "albumartist", "album_artist") }
func (t Tags) Genres() []string        { return t.getAllTagValues("genre") }
func (t Tags) Year() int               { return t.getYear("date") }
func (t Tags) Comment() string         { return t.getFirstTagValue("comment") }
func (t Tags) Lyrics() string          { return t.getFirstTagValue("lyrics", "lyrics-eng") }
func (t Tags) Compilation() bool       { return t.getBool("tcmp", "compilation") }
func (t Tags) TrackNumber() (int, int) { return t.getTuple("track", "tracknumber") }
func (t Tags) DiscNumber() (int, int)  { return t.getTuple("disc", "discnumber") }
func (t Tags) DiscSubtitle() string {
	return t.getFirstTagValue("tsst", "discsubtitle", "setsubtitle")
}
func (t Tags) CatalogNum() string { return t.getFirstTagValue("catalognumber") }
func (t Tags) Bpm() int           { return (int)(math.Round(t.getFloat("tbpm", "bpm", "fbpm"))) }
func (t Tags) HasPicture() bool   { return t.getFirstTagValue("has_picture") != "" }

// MusicBrainz Identifiers
func (t Tags) MbzReleaseTrackID() string {
	return t.getMbzID("musicbrainz_releasetrackid", "musicbrainz release track id")
}

func (t Tags) MbzTrackID() string { return t.getMbzID("musicbrainz_trackid", "musicbrainz track id") }
func (t Tags) MbzAlbumID() string { return t.getMbzID("musicbrainz_albumid", "musicbrainz album id") }
func (t Tags) MbzArtistID() string {
	return t.getMbzID("musicbrainz_artistid", "musicbrainz artist id")
}
func (t Tags) MbzAlbumArtistID() string {
	return t.getMbzID("musicbrainz_albumartistid", "musicbrainz album artist id")
}
func (t Tags) MbzAlbumType() string {
	return t.getFirstTagValue("musicbrainz_albumtype", "musicbrainz album type")
}
func (t Tags) MbzAlbumComment() string {
	return t.getFirstTagValue("musicbrainz_albumcomment", "musicbrainz album comment")
}

// File properties

func (t Tags) Duration() float32           { return float32(t.getFloat("duration")) }
func (t Tags) BitRate() int                { return t.getInt("bitrate") }
func (t Tags) Channels() int               { return t.getInt("channels") }
func (t Tags) ModificationTime() time.Time { return t.fileInfo.ModTime() }
func (t Tags) Size() int64                 { return t.fileInfo.Size() }
func (t Tags) FilePath() string            { return t.filePath }
func (t Tags) Suffix() string              { return strings.ToLower(strings.TrimPrefix(path.Ext(t.filePath), ".")) }

func (t Tags) getTags(tagNames ...string) []string {
	for _, tag := range tagNames {
		if v, ok := t.tags[tag]; ok {
			return v
		}
	}
	return nil
}

func (t Tags) getFirstTagValue(tagNames ...string) string {
	ts := t.getTags(tagNames...)
	if len(ts) > 0 {
		return ts[0]
	}
	return ""
}

func (t Tags) getAllTagValues(tagNames ...string) []string {
	var values []string
	for _, tag := range tagNames {
		if v, ok := t.tags[tag]; ok {
			values = append(values, v...)
		}
	}
	return values
}

func (t Tags) getSortTag(originalTag string, tagNamess ...string) string {
	formats := []string{"sort%s", "sort_%s", "sort-%s", "%ssort", "%s_sort", "%s-sort"}
	all := []string{originalTag}
	for _, tag := range tagNamess {
		for _, format := range formats {
			name := fmt.Sprintf(format, tag)
			all = append(all, name)
		}
	}
	return t.getFirstTagValue(all...)
}

var dateRegex = regexp.MustCompile(`([12]\d\d\d)`)

func (t Tags) getYear(tagNames ...string) int {
	tag := t.getFirstTagValue(tagNames...)
	if tag == "" {
		return 0
	}
	match := dateRegex.FindStringSubmatch(tag)
	if len(match) == 0 {
		log.Warn("Error parsing year date field", "file", t.filePath, "date", tag)
		return 0
	}
	year, _ := strconv.Atoi(match[1])
	return year
}

func (t Tags) getBool(tagNames ...string) bool {
	tag := t.getFirstTagValue(tagNames...)
	if tag == "" {
		return false
	}
	i, _ := strconv.Atoi(strings.TrimSpace(tag))
	return i == 1
}

func (t Tags) getTuple(tagNames ...string) (int, int) {
	tag := t.getFirstTagValue(tagNames...)
	if tag == "" {
		return 0, 0
	}
	tuple := strings.Split(tag, "/")
	t1, t2 := 0, 0
	t1, _ = strconv.Atoi(tuple[0])
	if len(tuple) > 1 {
		t2, _ = strconv.Atoi(tuple[1])
	} else {
		t2tag := t.getFirstTagValue(tagNames[0] + "total")
		t2, _ = strconv.Atoi(t2tag)
	}
	return t1, t2
}

func (t Tags) getMbzID(tagNames ...string) string {
	tag := t.getFirstTagValue(tagNames...)
	if _, err := uuid.Parse(tag); err != nil {
		return ""
	}
	return tag
}

func (t Tags) getInt(tagNames ...string) int {
	tag := t.getFirstTagValue(tagNames...)
	i, _ := strconv.Atoi(tag)
	return i
}

func (t Tags) getFloat(tagNames ...string) float64 {
	var tag = t.getFirstTagValue(tagNames...)
	var value, err = strconv.ParseFloat(tag, 64)
	if err != nil {
		return 0
	}
	return value
}
