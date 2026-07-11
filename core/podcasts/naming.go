package podcasts

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
)

// suffixByContentType maps common podcast enclosure content types to a file
// extension. Podcasts are almost always MP3, so that's the fallback.
var suffixByContentType = map[string]string{
	"audio/mpeg":      "mp3",
	"audio/mp3":       "mp3",
	"audio/mp4":       "m4a",
	"audio/x-m4a":     "m4a",
	"audio/aac":       "aac",
	"audio/ogg":       "ogg",
	"audio/vorbis":    "ogg",
	"audio/wav":       "wav",
	"audio/x-wav":     "wav",
	"audio/flac":      "flac",
	"audio/x-flac":    "flac",
	"audio/webm":      "weba",
	"video/mp4":       "mp4",
	"video/x-m4v":     "m4v",
	"video/quicktime": "mov",
}

const defaultAudioSuffix = "mp3"

// suffixFor picks a file extension for a downloaded episode, preferring the
// HTTP response's actual Content-Type over whatever the RSS feed advertised
// (feeds are frequently wrong about this).
func suffixFor(contentType, sourceUrl string) string {
	contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if s, ok := suffixByContentType[contentType]; ok {
		return s
	}
	if ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(sourceUrl)), "."); ext != "" && len(ext) <= 4 {
		return ext
	}
	return defaultAudioSuffix
}

// episodeStoragePath returns the path (relative to the podcasts storage
// root) a downloaded episode should live at. Using DB ids rather than
// sanitized RSS titles avoids collisions and path-traversal entirely.
func episodeStoragePath(ep model.PodcastEpisode, suffix string) string {
	return filepath.Join(ep.ChannelID, fmt.Sprintf("%s.%s", ep.ID, suffix))
}

// episodeAbsolutePath is the on-disk location for a (potentially
// not-yet-downloaded) episode's file, given a storage-relative path.
func episodeAbsolutePath(relativePath string) string {
	return filepath.Join(conf.Server.Podcasts.StorageFolder.String(), relativePath)
}
