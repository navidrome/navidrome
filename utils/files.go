package utils

import (
	"mime"
	"path/filepath"
	"strings"
)

var excludeAudioType = []string{
	"audio/x-mpegurl",
	"audio/x-scpls",
}

func IsAudioFile(filePath string) bool {
	extension := filepath.Ext(filePath)
	mimeType := mime.TypeByExtension(extension)
	return !StringInSlice(mimeType, excludeAudioType) && strings.HasPrefix(mimeType, "audio/")
}

func IsImageFile(filePath string) bool {
	extension := filepath.Ext(filePath)
	return strings.HasPrefix(mime.TypeByExtension(extension), "image/")
}
