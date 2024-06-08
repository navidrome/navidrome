package metadata

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/djherbis/times"
	"github.com/google/uuid"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type Extractor interface {
	Parse(files ...string) (map[string]ParsedTags, error)
	CustomMappings() ParsedTags
	Version() string
}

var extractors = map[string]Extractor{}

func RegisterExtractor(id string, parser Extractor) {
	extractors[id] = parser
}

func LogExtractors() {
	for id, p := range extractors {
		log.Debug("Registered metadata extractor", "id", id, "version", p.Version())
	}
}

func Extract(files ...string) (map[string]Tags, error) {
	p, ok := extractors[conf.Server.Scanner.Extractor]
	if !ok {
		log.Warn("Invalid 'Scanner.Extractor' option. Using default", "requested", conf.Server.Scanner.Extractor,
			"validOptions", "ffmpeg,taglib", "default", consts.DefaultScannerExtractor)
		p = extractors[consts.DefaultScannerExtractor]
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

		tags = tags.Map(p.CustomMappings())
		result[filePath] = NewTag(filePath, fileInfo, tags)
	}

	return result, nil
}

func NewTag(filePath string, fileInfo os.FileInfo, tags ParsedTags) Tags {
	for t, values := range tags {
		values = removeDuplicatesAndEmpty(values)
		if len(values) == 0 {
			delete(tags, t)
			continue
		}
		tags[t] = values
	}
	return Tags{
		filePath: filePath,
		fileInfo: fileInfo,
		Tags:     tags,
	}
}

func removeDuplicatesAndEmpty(values []string) []string {
	encountered := map[string]struct{}{}
	empty := true
	var result []string
	for _, v := range values {
		if _, ok := encountered[v]; ok {
			continue
		}
		encountered[v] = struct{}{}
		empty = empty && v == ""
		result = append(result, v)
	}
	if empty {
		return nil
	}
	return result
}

type ParsedTags map[string][]string

func (p ParsedTags) Map(customMappings ParsedTags) ParsedTags {
	if customMappings == nil {
		return p
	}
	for tagName, alternatives := range customMappings {
		for _, altName := range alternatives {
			if altValue, ok := p[altName]; ok {
				p[tagName] = append(p[tagName], altValue...)
				delete(p, altName)
			}
		}
	}
	return p
}

type Tags struct {
	filePath string
	fileInfo os.FileInfo
	Tags     ParsedTags
}

// Common tags

func (t Tags) Title() string  { return t.getFirstTagValue("title", "sort_name", "titlesort") }
func (t Tags) Album() string  { return t.getFirstTagValue("album", "sort_album", "albumsort") }
func (t Tags) Artist() string { return t.getFirstTagValue("artist", "sort_artist", "artistsort") }
func (t Tags) AlbumArtist() string {
	return t.getFirstTagValue("album_artist", "album artist", "albumartist")
}
func (t Tags) SortTitle() string           { return t.getSortTag("tsot", "title", "name") }
func (t Tags) SortAlbum() string           { return t.getSortTag("tsoa", "album") }
func (t Tags) SortArtist() string          { return t.getSortTag("tsop", "artist") }
func (t Tags) SortAlbumArtist() string     { return t.getSortTag("tso2", "albumartist", "album_artist") }
func (t Tags) Genres() []string            { return t.getAllTagValues("genre") }
func (t Tags) Date() (int, string)         { return t.getDate("date") }
func (t Tags) OriginalDate() (int, string) { return t.getDate("originaldate") }
func (t Tags) ReleaseDate() (int, string)  { return t.getDate("releasedate") }
func (t Tags) Comment() string             { return t.getFirstTagValue("comment") }
func (t Tags) Compilation() bool           { return t.getBool("tcmp", "compilation", "wm/iscompilation") }
func (t Tags) TrackNumber() (int, int)     { return t.getTuple("track", "tracknumber") }
func (t Tags) DiscNumber() (int, int)      { return t.getTuple("disc", "discnumber") }
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

func (t Tags) MbzRecordingID() string {
	return t.getMbzID("musicbrainz_trackid", "musicbrainz track id")
}
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

// Gain Properties

func (t Tags) RGAlbumGain() float64 {
	return t.getGainValue("replaygain_album_gain", "r128_album_gain")
}
func (t Tags) RGAlbumPeak() float64 { return t.getPeakValue("replaygain_album_peak") }
func (t Tags) RGTrackGain() float64 {
	return t.getGainValue("replaygain_track_gain", "r128_track_gain")
}
func (t Tags) RGTrackPeak() float64 { return t.getPeakValue("replaygain_track_peak") }

// File properties

func (t Tags) Duration() float32           { return float32(t.getFloat("duration")) }
func (t Tags) SampleRate() int             { return t.getInt("samplerate") }
func (t Tags) BitRate() int                { return t.getInt("bitrate") }
func (t Tags) Channels() int               { return t.getInt("channels") }
func (t Tags) ModificationTime() time.Time { return t.fileInfo.ModTime() }
func (t Tags) Size() int64                 { return t.fileInfo.Size() }
func (t Tags) FilePath() string            { return t.filePath }
func (t Tags) Suffix() string              { return strings.ToLower(strings.TrimPrefix(path.Ext(t.filePath), ".")) }
func (t Tags) BirthTime() time.Time {
	if ts := times.Get(t.fileInfo); ts.HasBirthTime() {
		return ts.BirthTime()
	}
	return time.Now()
}

func (t Tags) Lyrics() string {
	lyricList := model.LyricList{}
	basicLyrics := t.getAllTagValues("lyrics", "unsynced_lyrics", "unsynced lyrics", "unsyncedlyrics")

	for _, value := range basicLyrics {
		lyrics, err := model.ToLyrics("xxx", value)
		if err != nil {
			log.Warn("Unexpected failure occurred when parsing lyrics", "file", t.filePath, "error", err)
			continue
		}

		lyricList = append(lyricList, *lyrics)
	}

	for tag, value := range t.Tags {
		if strings.HasPrefix(tag, "lyrics-") {
			language := strings.TrimSpace(strings.TrimPrefix(tag, "lyrics-"))

			if language == "" {
				language = "xxx"
			}

			for _, text := range value {
				lyrics, err := model.ToLyrics(language, text)
				if err != nil {
					log.Warn("Unexpected failure occurred when parsing lyrics", "file", t.filePath, "error", err)
					continue
				}

				lyricList = append(lyricList, *lyrics)
			}
		}
	}

	res, err := json.Marshal(lyricList)
	if err != nil {
		log.Warn("Unexpected error occurred when serializing lyrics", "file", t.filePath, "error", err)
		return ""
	}
	return string(res)
}

func (t Tags) getGainValue(rgTagName, r128TagName string) float64 {
	// Check for ReplayGain first
	// ReplayGain is in the form [-]a.bb dB and normalized to -18dB
	var tag = t.getFirstTagValue(rgTagName)
	if tag != "" {
		tag = strings.TrimSpace(strings.Replace(tag, "dB", "", 1))
		var value, err = strconv.ParseFloat(tag, 64)
		if err != nil || value == math.Inf(-1) || value == math.Inf(1) {
			return 0
		}
		return value
	}

	// If ReplayGain is not found, check for R128 gain
	// R128 gain is a Q7.8 fixed point number normalized to -23dB
	tag = t.getFirstTagValue(r128TagName)
	if tag != "" {
		var iValue, err = strconv.Atoi(tag)
		if err != nil {
			return 0
		}
		// Convert Q7.8 to float
		var value = float64(iValue) / 256.0
		// Adding 5 dB to normalize with ReplayGain level
		return value + 5
	}

	return 0
}

func (t Tags) getPeakValue(tagName string) float64 {
	var tag = t.getFirstTagValue(tagName)
	var value, err = strconv.ParseFloat(tag, 64)
	if err != nil || value == math.Inf(-1) || value == math.Inf(1) {
		// A default of 1 for peak value results in no changes
		return 1
	}
	return value
}

func (t Tags) getTags(tagNames ...string) []string {
	for _, tag := range tagNames {
		if v, ok := t.Tags[tag]; ok {
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
		if v, ok := t.Tags[tag]; ok {
			values = append(values, v...)
		}
	}
	return values
}

func (t Tags) getSortTag(originalTag string, tagNames ...string) string {
	formats := []string{"sort%s", "sort_%s", "sort-%s", "%ssort", "%s_sort", "%s-sort"}
	all := []string{originalTag}
	for _, tag := range tagNames {
		for _, format := range formats {
			name := fmt.Sprintf(format, tag)
			all = append(all, name)
		}
	}
	return t.getFirstTagValue(all...)
}

var dateRegex = regexp.MustCompile(`([12]\d\d\d)`)

func (t Tags) getDate(tagNames ...string) (int, string) {
	tag := t.getFirstTagValue(tagNames...)
	if len(tag) < 4 {
		return 0, ""
	}
	// first get just the year
	match := dateRegex.FindStringSubmatch(tag)
	if len(match) == 0 {
		log.Warn("Error parsing "+tagNames[0]+" field for year", "file", t.filePath, "date", tag)
		return 0, ""
	}
	year, _ := strconv.Atoi(match[1])

	if len(tag) < 5 {
		return year, match[1]
	}

	//then try YYYY-MM-DD
	if len(tag) > 10 {
		tag = tag[:10]
	}
	layout := "2006-01-02"
	_, err := time.Parse(layout, tag)
	if err != nil {
		layout = "2006-01"
		_, err = time.Parse(layout, tag)
		if err != nil {
			log.Warn("Error parsing "+tagNames[0]+" field for month + day", "file", t.filePath, "date", tag)
			return year, match[1]
		}
	}
	return year, tag
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
