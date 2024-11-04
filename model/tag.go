package model

import (
	"cmp"
	"crypto/md5"
	"fmt"
	"slices"
	"strings"

	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/utils/slice"
)

type Tag struct {
	ID       string
	TagName  TagName
	TagValue string
}

type TagList []Tag

func (l TagList) GroupByFrequency() Tags {
	grouped := map[string]map[string]int{}
	for _, t := range l {
		if m, ok := grouped[string(t.TagName)]; !ok {
			grouped[string(t.TagName)] = map[string]int{t.TagValue: 0}
		} else {
			m[t.TagValue]++
		}
	}

	tags := Tags{}
	for name, values := range grouped {
		valueList := make([]string, 0, len(values))
		for value := range values {
			valueList = append(valueList, value)
		}
		slices.SortFunc(valueList, func(a, b string) int {
			return cmp.Or(
				cmp.Compare(values[b], values[a]),
				cmp.Compare(a, b),
			)
		})
		tags[TagName(name)] = valueList
	}
	return tags
}

func (t Tag) String() string {
	return fmt.Sprintf("%s=%s", t.TagName, t.TagValue)
}

func NewTag(name TagName, value string) Tag {
	name = name.ToLower()
	hashID := tagID(name, value)
	return Tag{
		ID:       hashID,
		TagName:  name,
		TagValue: value,
	}
}

func tagID(name TagName, value string) string {
	hashID := id.NewHash(string(name), strings.ToLower(value))
	return hashID
}

type Tags map[TagName][]string

func (t Tags) Values(name TagName) []string {
	return t[name]
}

func (t Tags) IDs() []string {
	var ids []string
	for name, tag := range t {
		name = name.ToLower()
		for _, v := range tag {
			ids = append(ids, tagID(name, strings.ToLower(v)))
		}
	}
	return ids
}

func (t Tags) Flatten(name TagName) TagList {
	var tags TagList
	for _, v := range t[name] {
		tags = append(tags, NewTag(name, v))
	}
	return tags
}

func (t Tags) FlattenAll() TagList {
	var tags TagList
	for name, values := range t {
		for _, v := range values {
			tags = append(tags, NewTag(name, v))
		}
	}
	return tags
}

func (t Tags) Sort() {
	for _, values := range t {
		slices.Sort(values)
	}
}

func (t Tags) Hash() []byte {
	if len(t) == 0 {
		return nil
	}
	ids := t.IDs()
	slices.Sort(ids)
	sum := md5.New()
	sum.Write([]byte(strings.Join(ids, "|")))
	return sum.Sum(nil)
}

func (t Tags) ToGenres() (string, Genres) {
	values := t.Values("genre")
	if len(values) == 0 {
		return "", nil
	}
	genres := slice.Map(values, func(g string) Genre {
		t := NewTag("genre", g)
		return Genre{ID: t.ID, Name: g}
	})
	return genres[0].Name, genres
}

// Merge merges the tags from another Tags object into this one, removing any duplicates
func (t Tags) Merge(tags Tags) {
	for name, values := range tags {
		for _, v := range values {
			t.Add(name, v)
		}
	}
}

func (t Tags) Add(name TagName, v string) {
	for _, existing := range t[name] {
		if existing == v {
			return
		}
	}
	t[name] = append(t[name], v)
}

type TagRepository interface {
	Add(...Tag) error
}

type TagName string

func (t TagName) ToLower() TagName {
	return TagName(strings.ToLower(string(t)))
}

func (t TagName) String() string {
	return string(t)
}

// Tag names, as defined in the mappings.yaml file
const (
	TagAlbum         TagName = "album"
	TagTitle         TagName = "title"
	TagTrackNumber   TagName = "track"
	TagDiscNumber    TagName = "disc"
	TagTotalTracks   TagName = "tracktotal"
	TagTotalDiscs    TagName = "disctotal"
	TagDiscSubtitle  TagName = "discsubtitle"
	TagGenre         TagName = "genre"
	TagMood          TagName = "mood"
	TagComment       TagName = "comment"
	TagAlbumSort     TagName = "albumsort"
	TagAlbumVersion  TagName = "albumversion"
	TagTitleSort     TagName = "titlesort"
	TagCompilation   TagName = "compilation"
	TagGrouping      TagName = "grouping"
	TagLyrics        TagName = "lyrics"
	TagRecordLabel   TagName = "recordlabel"
	TagReleaseType   TagName = "releasetype"
	TagMedia         TagName = "media"
	TagCatalogNumber TagName = "catalognumber"
	TagBPM           TagName = "bpm"

	// Dates and years

	TagOriginalDate  TagName = "originaldate"
	TagReleaseDate   TagName = "releasedate"
	TagRecordingDate TagName = "recordingdate"

	// Artists and roles

	TagAlbumArtist      TagName = "albumartist"
	TagAlbumArtists     TagName = "albumartists"
	TagAlbumArtistSort  TagName = "albumartistsort"
	TagAlbumArtistsSort TagName = "albumartistssort"
	TagTrackArtist      TagName = "artist"
	TagTrackArtists     TagName = "artists"
	TagTrackArtistSort  TagName = "artistsort"
	TagTrackArtistsSort TagName = "artistssort"
	TagComposer         TagName = "composer"
	TagComposerSort     TagName = "composersort"
	TagLyricist         TagName = "lyricist"
	TagLyricistSort     TagName = "lyricistsort"
	TagDirector         TagName = "director"
	TagProducer         TagName = "producer"
	TagEngineer         TagName = "engineer"
	TagMixer            TagName = "mixer"
	TagRemixer          TagName = "remixer"
	TagDJMixer          TagName = "djmixer"
	TagConductor        TagName = "conductor"
	TagArranger         TagName = "arranger"
	TagPerformer        TagName = "performer"

	// ReplayGain

	TagReplayGainAlbumGain TagName = "replaygain_album_gain"
	TagReplayGainAlbumPeak TagName = "replaygain_album_peak"
	TagReplayGainTrackGain TagName = "replaygain_track_gain"
	TagReplayGainTrackPeak TagName = "replaygain_track_peak"

	// MusicBrainz

	TagMusicBrainzArtistID       TagName = "musicbrainz_artistid"
	TagMusicBrainzRecordingID    TagName = "musicbrainz_recordingid"
	TagMusicBrainzTrackID        TagName = "musicbrainz_trackid"
	TagMusicBrainzAlbumArtistID  TagName = "musicbrainz_albumartistid"
	TagMusicBrainzAlbumID        TagName = "musicbrainz_albumid"
	TagMusicBrainzReleaseGroupID TagName = "musicbrainz_releasegroupid"
)
