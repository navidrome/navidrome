package metadata

import (
	"strings"
	"sync"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/resources"
	"gopkg.in/yaml.v3"
)

type Name string

// Tag names, as defined in the mappings.yaml file
const (
	Album         Name = "album"
	Title         Name = "title"
	TrackNumber   Name = "track"
	DiscNumber    Name = "disc"
	TotalTracks   Name = "tracktotal"
	TotalDiscs    Name = "disctotal"
	Genre         Name = "genre"
	Comment       Name = "comment"
	AlbumSort     Name = "albumsort"
	AlbumComment  Name = "albumcomment"
	TitleSort     Name = "titlesort"
	Compilation   Name = "compilation"
	Grouping      Name = "grouping"
	Lyrics        Name = "lyrics"
	RecordLabel   Name = "recordlabel"
	CatalogNumber Name = "catalognumber"
	BPM           Name = "bpm"

	// Dates and years

	OriginalDate Name = "originaldate"
	ReleaseDate  Name = "releasedate"

	// Artists and roles

	AlbumArtist     Name = "albumartist"
	AlbumArtistSort Name = "albumartistsort"
	TrackArtist     Name = "artist"
	TrackArtistSort Name = "artistsort"
	Composer        Name = "composer"
	ComposerSort    Name = "composersort"
	Performer       Name = "performer"
	Director        Name = "director"
	Conductor       Name = "conductor"
	Arranger        Name = "arranger"
	Lyricist        Name = "lyricist"

	// ReplayGain

	ReplayGainAlbumGain Name = "replaygain_album_gain"
	ReplayGainAlbumPeak Name = "replaygain_album_peak"
	ReplayGainTrackGain Name = "replaygain_track_gain"
	ReplayGainTrackPeak Name = "replaygain_track_peak"

	// MusicBrainz

	MusicBrainzArtistID      Name = "musicbrainz_artistid"
	MusicBrainzRecordingID   Name = "musicbrainz_recordingid"
	MusicBrainzTrackID       Name = "musicbrainz_trackid"
	MusicBrainzAlbumArtistID Name = "musicbrainz_albumartistid"
	MusicBrainzAlbumID       Name = "musicbrainz_albumid"
)

var mappings = sync.OnceValue(func() map[string][]string {
	mappingsFile, err := resources.FS().Open("mappings.yaml")
	if err != nil {
		log.Error("Error opening mappings.yaml", err)
	}
	decoder := yaml.NewDecoder(mappingsFile)
	var mappings map[string][]string
	err = decoder.Decode(&mappings)
	if err != nil {
		log.Error("Error decoding mappings.yaml", err)
	}
	normalized := map[string][]string{}
	for k, v := range mappings {
		k = strings.ToLower(k)
		for _, val := range v {
			normalized[k] = append(normalized[k], strings.ToLower(val))
		}
	}
	return normalized
})
