// Package gotaglib provides an alternative metadata extractor using go-taglib,
// a pure Go (WASM-based) implementation of TagLib.
//
// This extractor aims for parity with the CGO-based taglib extractor. It uses
// TagLib's PropertyMap interface for standard tags. The File handle API provides
// efficient access to format-specific tags (ID3v2 frames, MP4 atoms, ASF attributes)
// through a single file open operation.
//
// This extractor is registered under the name "gotaglib". It only works with a filesystem
// (fs.FS) and does not support direct local file paths. Files returned by the filesystem
// must implement io.ReadSeeker for go-taglib to read them.
package gotaglib

import (
	"errors"
	"io"
	"io/fs"
	"strings"
	"time"

	"github.com/navidrome/navidrome/core/storage/local"
	"github.com/navidrome/navidrome/model/metadata"
	"go.senan.xyz/taglib"
)

type extractor struct {
	fs fs.FS
}

func (e extractor) Parse(files ...string) (map[string]metadata.Info, error) {
	results := make(map[string]metadata.Info)
	for _, path := range files {
		props, err := e.extractMetadata(path)
		if err != nil {
			continue
		}
		results[path] = *props
	}
	return results, nil
}

func (e extractor) Version() string {
	return "go-taglib (TagLib 2.1.1 WASM)"
}

func (e extractor) extractMetadata(filePath string) (*metadata.Info, error) {
	f, close, err := e.openFile(filePath)
	if err != nil {
		return nil, err
	}
	defer close()

	// Get all tags and properties in one go
	allTags := f.AllTags()
	props := f.Properties()

	// Map properties to AudioProperties
	ap := metadata.AudioProperties{
		Duration:   props.Length.Round(time.Millisecond * 10),
		BitRate:    int(props.Bitrate),
		Channels:   int(props.Channels),
		SampleRate: int(props.SampleRate),
		BitDepth:   int(props.BitsPerSample),
	}

	// Convert normalized tags to lowercase keys (go-taglib returns UPPERCASE keys)
	normalizedTags := make(map[string][]string, len(allTags.Tags))
	for key, values := range allTags.Tags {
		lowerKey := strings.ToLower(key)
		normalizedTags[lowerKey] = values
	}

	// Process format-specific raw tags
	processRawTags(allTags, normalizedTags)

	// Parse track/disc totals from "N/Total" format
	parseTuple(normalizedTags, "track")
	parseTuple(normalizedTags, "disc")

	// Adjust some ID3 tags
	parseLyrics(normalizedTags)
	parseTIPL(normalizedTags)
	delete(normalizedTags, "tmcl") // TMCL is already parsed by TagLib

	// Determine if file has embedded picture
	hasPicture := len(props.Images) > 0

	return &metadata.Info{
		Tags:            normalizedTags,
		AudioProperties: ap,
		HasPicture:      hasPicture,
	}, nil
}

// openFile opens the file at filePath using the extractor's filesystem.
// It returns a TagLib File handle and a cleanup function to close resources.
func (e extractor) openFile(filePath string) (*taglib.File, func(), error) {
	// Open the file from the filesystem
	file, err := e.fs.Open(filePath)
	if err != nil {
		return nil, nil, err
	}
	rs, isSeekable := file.(io.ReadSeeker)
	if !isSeekable {
		file.Close()
		return nil, nil, errors.New("file is not seekable")
	}
	f, err := taglib.OpenStream(rs, taglib.WithReadStyle(taglib.ReadStyleFast))
	if err != nil {
		file.Close()
		return nil, nil, err
	}
	closeFunc := func() {
		f.Close()
		file.Close()
	}
	return f, closeFunc, nil
}

// parseTuple parses track/disc numbers in "N/Total" format and separates them.
// For example, tracknumber="2/10" becomes tracknumber="2" and tracktotal="10".
func parseTuple(tags map[string][]string, prop string) {
	tagName := prop + "number"
	tagTotal := prop + "total"
	if value, ok := tags[tagName]; ok && len(value) > 0 {
		parts := strings.Split(value[0], "/")
		tags[tagName] = []string{parts[0]}
		if len(parts) == 2 {
			tags[tagTotal] = []string{parts[1]}
		}
	}
}

// parseLyrics ensures lyrics tags have a language code.
// If lyrics exist without a language code, they are moved to "lyrics:xxx".
func parseLyrics(tags map[string][]string) {
	lyrics := tags["lyrics"]
	if len(lyrics) > 0 {
		tags["lyrics:xxx"] = lyrics
		delete(tags, "lyrics")
	}
}

// processRawTags processes format-specific raw tags based on the detected file format.
// This handles ID3v2 frames (MP3/WAV/AIFF), MP4 atoms, and ASF attributes.
func processRawTags(allTags taglib.AllTags, normalizedTags map[string][]string) {
	switch allTags.Format {
	case taglib.FormatMPEG, taglib.FormatWAV, taglib.FormatAIFF:
		parseID3v2Frames(allTags.Raw, normalizedTags)
	case taglib.FormatMP4:
		parseMP4Atoms(allTags.Raw, normalizedTags)
	case taglib.FormatASF:
		parseASFAttributes(allTags.Raw, normalizedTags)
	}
}

// parseID3v2Frames processes ID3v2 raw frames to extract USLT/SYLT with language codes.
// This extracts language-specific lyrics that the standard Tags() doesn't provide.
func parseID3v2Frames(rawFrames map[string][]string, tags map[string][]string) {
	// Process frames that have language-specific data
	for key, values := range rawFrames {
		lowerKey := strings.ToLower(key)

		// Handle USLT:xxx and SYLT:xxx (lyrics with language codes)
		if strings.HasPrefix(lowerKey, "uslt:") || strings.HasPrefix(lowerKey, "sylt:") {
			parts := strings.SplitN(lowerKey, ":", 2)
			if len(parts) == 2 && parts[1] != "" {
				lang := parts[1]
				lyricsKey := "lyrics:" + lang
				tags[lyricsKey] = append(tags[lyricsKey], values...)
			}
		}
	}

	// If we found any language-specific lyrics from ID3v2 frames, remove the generic lyrics
	for key := range tags {
		if strings.HasPrefix(key, "lyrics:") && key != "lyrics" {
			delete(tags, "lyrics")
			break
		}
	}
}

const iTunesKeyPrefix = "----:com.apple.iTunes:"

// parseMP4Atoms processes MP4 raw atoms to get iTunes-specific tags.
func parseMP4Atoms(rawAtoms map[string][]string, tags map[string][]string) {
	// Process all atoms and add them to tags
	for key, values := range rawAtoms {
		// Strip iTunes prefix and convert to lowercase
		normalizedKey := strings.TrimPrefix(key, iTunesKeyPrefix)
		normalizedKey = strings.ToLower(normalizedKey)

		// Only add if the tag doesn't already exist (avoid duplication with PropertyMap)
		if _, exists := tags[normalizedKey]; !exists {
			tags[normalizedKey] = values
		}
	}
}

// parseASFAttributes processes ASF raw attributes to get WMA-specific tags.
func parseASFAttributes(rawAttrs map[string][]string, tags map[string][]string) {
	// Process all attributes and add them to tags
	for key, values := range rawAttrs {
		normalizedKey := strings.ToLower(key)

		// Only add if the tag doesn't already exist (avoid duplication with PropertyMap)
		if _, exists := tags[normalizedKey]; !exists {
			tags[normalizedKey] = values
		}
	}
}

// These are the only roles we support, based on Picard's tag map:
// https://picard-docs.musicbrainz.org/downloads/MusicBrainz_Picard_Tag_Map.html
var tiplMapping = map[string]string{
	"arranger": "arranger",
	"engineer": "engineer",
	"producer": "producer",
	"mix":      "mixer",
	"DJ-mix":   "djmixer",
}

// parseTIPL parses the ID3v2.4 TIPL frame string, which is received from TagLib in the format:
//
//	"arranger Andrew Powell engineer Chris Blair engineer Pat Stapley producer Eric Woolfson".
//
// and breaks it down into a map of roles and names, e.g.:
//
//	{"arranger": ["Andrew Powell"], "engineer": ["Chris Blair", "Pat Stapley"], "producer": ["Eric Woolfson"]}.
func parseTIPL(tags map[string][]string) {
	tipl := tags["tipl"]
	if len(tipl) == 0 {
		return
	}
	addRole := func(currentRole string, currentValue []string) {
		if currentRole != "" && len(currentValue) > 0 {
			role := tiplMapping[currentRole]
			tags[role] = append(tags[role], strings.Join(currentValue, " "))
		}
	}
	var currentRole string
	var currentValue []string
	for _, part := range strings.Split(tipl[0], " ") {
		if _, ok := tiplMapping[part]; ok {
			addRole(currentRole, currentValue)
			currentRole = part
			currentValue = nil
			continue
		}
		currentValue = append(currentValue, part)
	}
	addRole(currentRole, currentValue)
	delete(tags, "tipl")
}

var _ local.Extractor = (*extractor)(nil)

func init() {
	local.RegisterExtractor("taglib", func(fsys fs.FS, baseDir string) local.Extractor {
		return &extractor{fsys}
	})
}
