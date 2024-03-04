package cuesheet

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/navidrome/navidrome/scanner/metadata"
)

type TagsExtractor struct {
	md       *metadata.Tags
	fileInfo os.FileInfo
	result   map[int]metadata.ParsedTags
}

func NewExtractor(md *metadata.Tags) (*TagsExtractor, error) {
	fileInfo, err := os.Stat(md.FilePath())
	if err != nil {
		return nil, err
	}
	if md == nil {
		md = &metadata.Tags{}
	}
	return &TagsExtractor{
		md:       md,
		fileInfo: fileInfo,
		result:   map[int]metadata.ParsedTags{},
	}, nil
}

func makeDuration(d time.Duration) string {
	return strconv.FormatFloat(d.Seconds(), 'f', 3, 32)
}

func (e *TagsExtractor) Extract(cuesheet *Cuesheet, last bool, embedded bool) error {
	for _, file := range cuesheet.File {
		if !embedded && file.FileName != filepath.Base(e.md.FilePath()) {
			continue
		}
		totalTracks := len(file.Tracks)
		lastTrack := totalTracks - 1
		for i, t := range file.Tracks {
			tags := e.result[i]
			if tags == nil {
				tags = metadata.ParsedTags{}
				e.result[i] = tags
			}
			addTagValue("title", tags, t.Title)
			addTagValue("sub_track", tags, strconv.Itoa(i))
			if last {
				addTagValue("artist", tags, t.Performer, cuesheet.Performer, e.md.Artist())
				addTagValue("album", tags, cuesheet.Title, e.md.Album())
				addTagValue("genre", tags, cuesheet.Rem.Genre())
				tags["genre"] = append(tags["genre"], e.md.Genres()...)
				addTagValue("musicbrainz_albumid", tags, e.md.MbzAlbumID())
				addTagValue("musicbrainz_artistid", tags, e.md.MbzArtistID())
				addTagValue("musicbrainz_trackid", tags, e.md.MbzRecordingID())
				addTagValue("musicbrainz_releasetrackid", tags, e.md.MbzReleaseTrackID())
				disc, disctotal := e.md.DiscNumber()
				if disc > 0 && disctotal > 0 {
					addTagValue("discnumber", tags, strconv.FormatUint(uint64(disc), 10))
					addTagValue("discnumbertotal", tags, strconv.FormatUint(uint64(disctotal), 10))
				}
			} else {
				addTagValue("artist", tags, t.Performer, cuesheet.Performer)
				addTagValue("album", tags, cuesheet.Title)
				addTagValue("genre", tags, cuesheet.Rem.Genre())
				if t.Rem.TotalDiscs() > 0 {
					if t.Rem.DiscNumber() > 0 {
						addTagValue("discnumber", tags, strconv.FormatUint(uint64(t.Rem.DiscNumber()), 10))
					}
					addTagValue("discnumbertotal", tags, strconv.FormatUint(uint64(t.Rem.TotalDiscs()), 10))
				}
			}
			addTagValue("album_artist", tags, cuesheet.Performer)
			addTagValue("comment", tags, t.Rem.Comment(), cuesheet.Rem.Comment())
			addTagValue("tracknumber", tags, strconv.FormatUint(uint64(t.TrackNumber), 10))
			addTagValue("tracknumbertotal", tags, strconv.FormatUint(uint64(totalTracks), 10))
			addTagValue("date", tags, cuesheet.Rem.Date())
			addTagValue("replaygain_album_gain", tags, cuesheet.Rem.AlbumGain())
			addTagValue("replaygain_album_peak", tags, cuesheet.Rem.AlbumPeak())
			addTagValue("replaygain_track_gain", tags, t.Rem.TrackGain())
			addTagValue("replaygain_track_peak", tags, t.Rem.TrackPeak())
			addTagValue("bitrate", tags, fmt.Sprintf("%d", e.md.BitRate()))
			addTagValue("channels", tags, fmt.Sprintf("%d", e.md.Channels()))
			addTagValue("offset", tags, makeDuration(t.GetStartOffset()))
			var length time.Duration
			if i < lastTrack {
				nextTrack := file.Tracks[i+1]
				length = nextTrack.GetStartOffset() - t.GetStartOffset()
			} else {
				fileDuration := time.Millisecond * time.Duration(e.md.Duration()*1000.0)
				length = fileDuration - t.GetStartOffset()
			}
			if length < 0 {
				length = 0
			}
			addTagValue("duration", tags, makeDuration(length))
		}
	}

	return nil
}

func (e *TagsExtractor) ForEachTrack(trackCallback func(metadata.Tags)) {
	for _, md := range e.result {
		trackCallback(metadata.NewTag(e.md.FilePath(), e.fileInfo, md))
	}
}

func addTagValue(name string, tags metadata.ParsedTags, values ...string) {
	for _, v := range values {
		if v != "" {
			tags[name] = append(tags[name], v)
			return
		}
	}
}
