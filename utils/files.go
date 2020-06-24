package utils

import (
	"mime"
	"path/filepath"
	"strings"
)

func IsAudioFile(filePath string) bool {
	extension := filepath.Ext(filePath)
	return strings.HasPrefix(mime.TypeByExtension(extension), "audio/")
}

func IsImageFile(filePath string) bool {
	extension := filepath.Ext(filePath)
	return strings.HasPrefix(mime.TypeByExtension(extension), "image/")
}
