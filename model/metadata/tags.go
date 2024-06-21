package metadata

import (
	"strings"
	"sync"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/resources"
	"gopkg.in/yaml.v3"
)

type TagName string

// Tag names, as defined in the mappings.yaml file
const (
	Album         TagName = "album"
	Title         TagName = "title"
	TrackNumber   TagName = "track"
	DiscNumber    TagName = "disc"
	TotalTracks   TagName = "tracktotal"
	TotalDiscs    TagName = "disctotal"
	DiscSubtitle  TagName = "discsubtitle"
	Genre         TagName = "genre"
	Comment       TagName = "comment"
	AlbumSort     TagName = "albumsort"
	AlbumVersion  TagName = "albumversion"
	TitleSort     TagName = "titlesort"
	Compilation   TagName = "compilation"
	Grouping      TagName = "grouping"
	Lyrics        TagName = "lyrics"
	RecordLabel   TagName = "recordlabel"
	CatalogNumber TagName = "catalognumber"
	BPM           TagName = "bpm"

	// Dates and years

	OriginalDate  TagName = "originaldate"
	ReleaseDate   TagName = "releasedate"
	RecordingDate TagName = "recordingdate"

	// Artists and roles

	AlbumArtist      TagName = "albumartist"
	AlbumArtists     TagName = "albumartists"
	AlbumArtistSort  TagName = "albumartistsort"
	AlbumArtistsSort TagName = "albumartistssort"
	TrackArtist      TagName = "artist"
	TrackArtists     TagName = "artists"
	TrackArtistSort  TagName = "artistsort"
	TrackArtistsSort TagName = "artistssort"
	Composer         TagName = "composer"
	ComposerSort     TagName = "composersort"
	Lyricist         TagName = "lyricist"
	LyricistSort     TagName = "lyricistsort"
	Director         TagName = "director"
	Producer         TagName = "producer"
	Engineer         TagName = "engineer"
	Mixer            TagName = "mixer"
	Remixer          TagName = "remixer"
	DJMixer          TagName = "djmixer"
	Conductor        TagName = "conductor"
	Arranger         TagName = "arranger"
	Performer        TagName = "performer"

	// ReplayGain

	ReplayGainAlbumGain TagName = "replaygain_album_gain"
	ReplayGainAlbumPeak TagName = "replaygain_album_peak"
	ReplayGainTrackGain TagName = "replaygain_track_gain"
	ReplayGainTrackPeak TagName = "replaygain_track_peak"

	// MusicBrainz

	MusicBrainzArtistID      TagName = "musicbrainz_artistid"
	MusicBrainzRecordingID   TagName = "musicbrainz_recordingid"
	MusicBrainzTrackID       TagName = "musicbrainz_trackid"
	MusicBrainzAlbumArtistID TagName = "musicbrainz_albumartistid"
	MusicBrainzAlbumID       TagName = "musicbrainz_albumid"
)

type tagMapping struct {
	Aliases   []string `yaml:"aliases"`
	Type      TagType  `yaml:"type"`
	MaxLength int      `yaml:"maxLength"`
}

type TagType string

const (
	TagTypeInteger TagType = "integer"
	TagTypeFloat   TagType = "float"
	TagTypeDate    TagType = "date"
	TagTypeUUID    TagType = "uuid"
)

var mappings = sync.OnceValue(func() map[string]tagMapping {
	mappingsFile, err := resources.FS().Open("mappings.yaml")
	if err != nil {
		log.Error("Error opening mappings.yaml", err)
	}
	decoder := yaml.NewDecoder(mappingsFile)
	var mappings map[string]tagMapping
	err = decoder.Decode(&mappings)
	if err != nil {
		log.Error("Error decoding mappings.yaml", err)
	}
	normalized := map[string]tagMapping{}
	for k, v := range mappings {
		k = strings.ToLower(k)
		var aliases []string
		for _, val := range v.Aliases {
			aliases = append(aliases, strings.ToLower(val))
		}
		normalized[k] = tagMapping{Aliases: aliases, Type: v.Type, MaxLength: v.MaxLength}
	}
	return normalized
})
