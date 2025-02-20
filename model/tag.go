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
	ID             string  `json:"id,omitempty"`
	TagName        TagName `json:"tagName,omitempty"`
	TagValue       string  `json:"tagValue,omitempty"`
	AlbumCount     int     `json:"albumCount,omitempty"`
	MediaFileCount int     `json:"songCount,omitempty"`
}

type TagList []Tag

func (l TagList) GroupByFrequency() Tags {
	grouped := map[string]map[string]int{}
	values := map[string]string{}
	for _, t := range l {
		if m, ok := grouped[string(t.TagName)]; !ok {
			grouped[string(t.TagName)] = map[string]int{t.ID: 1}
		} else {
			m[t.ID]++
		}
		values[t.ID] = t.TagValue
	}

	tags := Tags{}
	for name, counts := range grouped {
		idList := make([]string, 0, len(counts))
		for tid := range counts {
			idList = append(idList, tid)
		}
		slices.SortFunc(idList, func(a, b string) int {
			return cmp.Or(
				cmp.Compare(counts[b], counts[a]),
				cmp.Compare(values[a], values[b]),
			)
		})
		tags[TagName(name)] = slice.Map(idList, func(id string) string { return values[id] })
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
	return id.NewTagID(string(name), value)
}

type RawTags map[string][]string

type Tags map[TagName][]string

func (t Tags) Values(name TagName) []string {
	return t[name]
}

func (t Tags) IDs() []string {
	var ids []string
	for name, tag := range t {
		name = name.ToLower()
		for _, v := range tag {
			ids = append(ids, tagID(name, v))
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
	UpdateCounts() error
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
	TagAlbum          TagName = "album"
	TagTitle          TagName = "title"
	TagTrackNumber    TagName = "track"
	TagDiscNumber     TagName = "disc"
	TagTotalTracks    TagName = "tracktotal"
	TagTotalDiscs     TagName = "disctotal"
	TagDiscSubtitle   TagName = "discsubtitle"
	TagSubtitle       TagName = "subtitle"
	TagGenre          TagName = "genre"
	TagMood           TagName = "mood"
	TagComment        TagName = "comment"
	TagAlbumSort      TagName = "albumsort"
	TagAlbumVersion   TagName = "albumversion"
	TagTitleSort      TagName = "titlesort"
	TagCompilation    TagName = "compilation"
	TagGrouping       TagName = "grouping"
	TagLyrics         TagName = "lyrics"
	TagRecordLabel    TagName = "recordlabel"
	TagReleaseType    TagName = "releasetype"
	TagReleaseCountry TagName = "releasecountry"
	TagMedia          TagName = "media"
	TagCatalogNumber  TagName = "catalognumber"
	TagBPM            TagName = "bpm"
	TagExplicitStatus TagName = "explicitstatus"

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
	TagR128AlbumGain       TagName = "r128_album_gain"
	TagR128TrackGain       TagName = "r128_track_gain"

	// MusicBrainz

	TagMusicBrainzArtistID       TagName = "musicbrainz_artistid"
	TagMusicBrainzRecordingID    TagName = "musicbrainz_recordingid"
	TagMusicBrainzTrackID        TagName = "musicbrainz_trackid"
	TagMusicBrainzAlbumArtistID  TagName = "musicbrainz_albumartistid"
	TagMusicBrainzAlbumID        TagName = "musicbrainz_albumid"
	TagMusicBrainzReleaseGroupID TagName = "musicbrainz_releasegroupid"

	TagMusicBrainzComposerID  TagName = "musicbrainz_composerid"
	TagMusicBrainzLyricistID  TagName = "musicbrainz_lyricistid"
	TagMusicBrainzDirectorID  TagName = "musicbrainz_directorid"
	TagMusicBrainzProducerID  TagName = "musicbrainz_producerid"
	TagMusicBrainzEngineerID  TagName = "musicbrainz_engineerid"
	TagMusicBrainzMixerID     TagName = "musicbrainz_mixerid"
	TagMusicBrainzRemixerID   TagName = "musicbrainz_remixerid"
	TagMusicBrainzDJMixerID   TagName = "musicbrainz_djmixerid"
	TagMusicBrainzConductorID TagName = "musicbrainz_conductorid"
	TagMusicBrainzArrangerID  TagName = "musicbrainz_arrangerid"
	TagMusicBrainzPerformerID TagName = "musicbrainz_performerid"
)
