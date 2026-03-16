package model

import (
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
)

// UploadedImagePath returns the absolute filesystem path for a manually uploaded
// entity cover image. Returns empty string if filename is empty.
func UploadedImagePath(entityType, filename string) string {
	if filename == "" {
		return ""
	}
	return filepath.Join(conf.Server.DataFolder, consts.ArtworkFolder, entityType, filename)
}
