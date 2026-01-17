// Package gotaglib provides an alternative metadata extractor using go-taglib,
// a pure Go (WASM-based) implementation of TagLib.
//
// This extractor aims for parity with the CGO-based taglib extractor. It uses
// TagLib's PropertyMap interface for standard tags, ReadID3v2Frames for
// ID3v2-specific frames like USLT and SYLT with proper language codes, and
// ReadMP4Atoms for iTunes-specific M4A tags.
//
// Known Limitations:
//
//   - BitDepth: Not available. go-taglib's WASM module only exposes generic audio
//     properties (length, channels, sampleRate, bitrate), not format-specific
//     properties like bitsPerSample. MediaFile.BitDepth will always be 0.
//
//   - WMA/ASF specific tags: Some ASF-specific tags (like replaygain) may not be
//     available. The CGO extractor reads from asfFile->tag()->attributeListMap().
//
// For full feature parity, use the CGO-based taglib extractor. This extractor
// is provided for environments where CGO is not available (e.g., cross-compilation).
package gotaglib

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/navidrome/navidrome/core/storage/local"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/metadata"
	"go.senan.xyz/taglib"
)

type extractor struct {
	baseDir string
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
	fullPath := filepath.Join(e.baseDir, filePath)
	// Read tags
	tags, err := taglib.ReadTags(fullPath)
	if err != nil {
		// Check if file doesn't exist
		if _, statErr := os.Stat(fullPath); os.IsNotExist(statErr) {
			return nil, fs.ErrNotExist
		}
		// Check if permission denied
		if errors.Is(err, taglib.ErrInvalidFile) {
			// Try to open the file to check for permission errors
			if f, openErr := os.Open(fullPath); openErr != nil {
				if os.IsPermission(openErr) {
					return nil, os.ErrPermission
				}
			} else {
				f.Close()
			}
		}
		log.Warn("gotaglib extractor: Error reading metadata from file. Skipping", "filePath", fullPath, err)
		return nil, err
	}

	// Read audio properties
	props, err := taglib.ReadProperties(fullPath)
	if err != nil {
		log.Warn("gotaglib extractor: Error reading properties from file. Skipping", "filePath", fullPath, err)
		return nil, err
	}

	// Map properties to AudioProperties
	ap := metadata.AudioProperties{
		Duration:   props.Length.Round(time.Millisecond * 10),
		BitRate:    int(props.Bitrate),
		Channels:   int(props.Channels),
		SampleRate: int(props.SampleRate),
		// Note: go-taglib doesn't expose bit depth directly in Properties
		// BitDepth will be 0 for formats where it's not available
	}

	// Convert tags to lowercase keys (go-taglib returns UPPERCASE keys)
	normalizedTags := make(map[string][]string)
	for key, values := range tags {
		lowerKey := strings.ToLower(key)
		normalizedTags[lowerKey] = values
	}

	// For MP3 files, read ID3v2 frames directly to get USLT/SYLT with language codes
	ext := strings.ToLower(filepath.Ext(fullPath))
	if ext == ".mp3" {
		parseID3v2Frames(fullPath, normalizedTags)
	}

	// For M4A files, read MP4 atoms for iTunes-specific tags
	if ext == ".m4a" {
		parseMP4Atoms(fullPath, normalizedTags)
	}

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

// parseID3v2Frames reads ID3v2 frames directly to get USLT/SYLT with language codes.
// This extracts language-specific lyrics that the standard ReadTags doesn't provide.
func parseID3v2Frames(fullPath string, tags map[string][]string) {
	frames, err := taglib.ReadID3v2Frames(fullPath)
	if err != nil {
		return
	}

	// Process frames that have language-specific data
	for key, values := range frames {
		lowerKey := strings.ToLower(key)

		// Handle USLT:xxx and SYLT:xxx (lyrics with language codes)
		if strings.HasPrefix(lowerKey, "uslt:") || strings.HasPrefix(lowerKey, "sylt:") {
			lang := strings.TrimPrefix(lowerKey, "uslt:")
			lang = strings.TrimPrefix(lang, "sylt:")
			if lang != "" {
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

// parseMP4Atoms reads MP4 atoms directly to get iTunes-specific tags.
func parseMP4Atoms(fullPath string, tags map[string][]string) {
	atoms, err := taglib.ReadMP4Atoms(fullPath)
	if err != nil {
		return
	}

	// Process all atoms and add them to tags
	for key, values := range atoms {
		// Strip iTunes prefix and convert to lowercase
		normalizedKey := strings.TrimPrefix(key, iTunesKeyPrefix)
		normalizedKey = strings.ToLower(normalizedKey)

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
	local.RegisterExtractor("gotaglib", func(_ fs.FS, baseDir string) local.Extractor {
		return &extractor{baseDir}
	})
}
