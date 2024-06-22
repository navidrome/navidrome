package model

import (
	"cmp"
	"crypto/md5"
	"fmt"
	"slices"
	"strings"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/utils/slice"
)

type Tag struct {
	ID       string
	TagName  TagName
	TagValue string
}

type TagList []Tag

func (t Tag) String() string {
	return fmt.Sprintf("%s=%s", t.TagName, t.TagValue)
}

func NewTag(name TagName, value string) Tag {
	name = name.ToLower()
	id := fmt.Sprintf("%x", md5.Sum([]byte(string(name)+consts.Zwsp+strings.ToLower(value))))
	return Tag{
		ID:       id,
		TagName:  name,
		TagValue: value,
	}
}

type Tags map[TagName][]string

func (t Tags) Values(name TagName) []string {
	return t[name]
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

func (t Tags) Hash() string {
	if len(t) == 0 {
		return ""
	}
	all := t.FlattenAll()
	slices.SortFunc(all, func(a, b Tag) int {
		return cmp.Compare(a.ID, b.ID)
	})
	sum := md5.New()
	for _, tag := range all {
		sum.Write([]byte(tag.ID))
	}
	return fmt.Sprintf("%x", sum.Sum(nil))
}

func (t Tags) ToGenres() (string, Genres) {
	var genres Genres
	for _, g := range t.Values("genre") {
		t := NewTag("genre", g)
		genres = append(genres, Genre{ID: t.ID, Name: g})
	}
	// TODO This will not work, as there is only one instance of each genre in the tags
	return slice.MostFrequent(t.Values("genre")), genres
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

	TagMusicBrainzArtistID      TagName = "musicbrainz_artistid"
	TagMusicBrainzRecordingID   TagName = "musicbrainz_recordingid"
	TagMusicBrainzTrackID       TagName = "musicbrainz_trackid"
	TagMusicBrainzAlbumArtistID TagName = "musicbrainz_albumartistid"
	TagMusicBrainzAlbumID       TagName = "musicbrainz_albumid"
)
