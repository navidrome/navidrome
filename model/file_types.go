package model

import (
	"mime"
	"path/filepath"
	"slices"
	"strings"
)

var excludeAudioType = []string{
	"audio/x-mpegurl",
	"audio/x-scpls",
}

func IsAudioFile(filePath string) bool {
	extension := filepath.Ext(filePath)
	mimeType := mime.TypeByExtension(extension)
	return !slices.Contains(excludeAudioType, mimeType) && strings.HasPrefix(mimeType, "audio/")
}

func IsCueSheetFile(filePath string) bool {
	extension := strings.ToLower(filepath.Ext(filePath))
	return extension == ".cue"
}

func IsImageFile(filePath string) bool {
	extension := filepath.Ext(filePath)
	return strings.HasPrefix(mime.TypeByExtension(extension), "image/")
}

func IsValidPlaylist(filePath string) bool {
	extension := strings.ToLower(filepath.Ext(filePath))
	return extension == ".m3u" || extension == ".m3u8" || extension == ".nsp"
}
