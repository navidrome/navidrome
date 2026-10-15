package artwork

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/dustin/go-humanize"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
)

// MaxImageUploadSize returns the configured max upload size in bytes, or the built-in default.
func MaxImageUploadSize() int64 {
	return parseSize(conf.Server.MaxImageUploadSize, consts.DefaultMaxImageUploadSize)
}

func parseSize(value, fallback string) int64 {
	if size, err := humanize.ParseBytes(value); err == nil && size > 0 {
		return int64(size)
	}
	size, _ := humanize.ParseBytes(fallback)
	return int64(size)
}

// Uploader stores a user-uploaded entity image and invalidates that entity's artwork state.
type Uploader interface {
	SetImage(ctx context.Context, entityType string, entityID string, name string, oldPath string, reader io.Reader, ext string) (filename string, err error)
	RemoveImage(ctx context.Context, path string) error
	// EnqueueArtwork re-resolves the item's artwork. Call it AFTER persisting the new
	// filename, or the worker resolves the old one.
	EnqueueArtwork(ctx context.Context, entityType, entityID string)
}

var uploadEntityKind = map[string]model.Kind{
	consts.EntityArtist:   model.KindArtistArtwork,
	consts.EntityPlaylist: model.KindPlaylistArtwork,
	consts.EntityRadio:    model.KindRadioArtwork,
}

type uploader struct {
	ds model.DataStore
}

func NewUploader(ds model.DataStore) Uploader {
	return &uploader{ds: ds}
}

func (s *uploader) SetImage(ctx context.Context, entityType string, entityID string, name string, oldPath string, reader io.Reader, ext string) (string, error) {
	filename := imageFilename(entityID, name, ext)
	absPath := model.UploadedImagePath(entityType, filename)

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return "", fmt.Errorf("creating image directory: %w", err)
	}

	if oldPath != "" {
		if err := os.Remove(oldPath); err != nil && !os.IsNotExist(err) {
			log.Warn(ctx, "Artwork: Failed to remove old image", "path", oldPath, err)
		}
	}

	f, err := os.Create(absPath)
	if err != nil {
		return "", fmt.Errorf("creating image file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, reader); err != nil {
		return "", fmt.Errorf("writing image file: %w", err)
	}
	return filename, nil
}

func (s *uploader) EnqueueArtwork(ctx context.Context, entityType, id string) {
	kind, ok := uploadEntityKind[entityType]
	if !ok {
		return
	}
	if err := Refresh(ctx, s.ds, kind, id); err != nil {
		log.Warn(ctx, "Artwork: Could not refresh artwork after upload", "kind", kind, "id", id, err)
	}
}

func (s *uploader) RemoveImage(ctx context.Context, path string) error {
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing image %q: %w", path, err)
	}
	return nil
}

func imageFilename(id, name, ext string) string {
	clean := utils.CleanFileName(name)
	if clean == "" {
		return id + ext
	}
	return id + "_" + clean + ext
}
