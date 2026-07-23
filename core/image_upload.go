package core

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

type ImageUploadService interface {
	SetImage(ctx context.Context, entityType string, entityID string, name string, oldPath string, reader io.Reader, ext string) (filename string, err error)
	RemoveImage(ctx context.Context, path string) error
	// EnqueueArtwork clears an item's resolved state and re-queues it at Bump priority. Callers
	// must invoke it AFTER persisting the new filename, so the worker never resolves the old one.
	EnqueueArtwork(ctx context.Context, entityType, entityID string)
}

// MaxImageUploadSize returns the configured MaxImageUploadSize in bytes, or the built-in default
// when it's unset/invalid. Shared by every API that accepts image uploads.
func MaxImageUploadSize() int64 {
	if size, err := humanize.ParseBytes(conf.Server.MaxImageUploadSize); err == nil && size > 0 {
		return int64(size)
	}
	size, _ := humanize.ParseBytes(consts.DefaultMaxImageUploadSize)
	return int64(size)
}

// uploadEntityKind maps an upload's entity type to its artwork kind prefix, so a
// successful upload can clear and re-queue that item's artwork state.
var uploadEntityKind = map[string]string{
	consts.EntityArtist:   model.KindArtistArtwork.Prefix(),
	consts.EntityPlaylist: model.KindPlaylistArtwork.Prefix(),
	consts.EntityRadio:    model.KindRadioArtwork.Prefix(),
}

type imageUploadService struct {
	ds model.DataStore
}

func NewImageUploadService(ds model.DataStore) ImageUploadService {
	return &imageUploadService{ds: ds}
}

func (s *imageUploadService) SetImage(ctx context.Context, entityType string, entityID string, name string, oldPath string, reader io.Reader, ext string) (string, error) {
	filename := imageFilename(entityID, name, ext)
	absPath := model.UploadedImagePath(entityType, filename)

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return "", fmt.Errorf("creating image directory: %w", err)
	}

	// Remove old image if it exists
	if oldPath != "" {
		if err := os.Remove(oldPath); err != nil && !os.IsNotExist(err) {
			log.Warn(ctx, "Failed to remove old image", "path", oldPath, err)
		}
	}

	// Save new image
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

// EnqueueArtwork clears the item's resolved state and re-queues it at Bump priority: the
// upload is now the top-priority source, so the worker re-resolves and the UI swaps.
func (s *imageUploadService) EnqueueArtwork(ctx context.Context, entityType, id string) {
	kind, ok := uploadEntityKind[entityType]
	if !ok {
		return
	}
	if err := s.ds.Artwork(ctx).DeleteForItem(kind, id); err != nil {
		log.Warn(ctx, "Could not clear artwork state after upload", "kind", kind, "id", id, err)
	}
	item := model.ArtworkQueueItem{ItemKind: kind, ItemID: id, ImageType: model.ImageTypePrimary,
		Priority: model.ArtworkPriorityBump}
	if err := s.ds.ArtworkQueue(ctx).Enqueue(item); err != nil {
		log.Warn(ctx, "Could not enqueue artwork after upload", "kind", kind, "id", id, err)
	}
}

func (s *imageUploadService) RemoveImage(ctx context.Context, path string) error {
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
