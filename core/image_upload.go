package core

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
)

type ImageUploadService interface {
	SetImage(ctx context.Context, entityType string, entityID string, name string, oldPath string, reader io.Reader, ext string) (filename string, err error)
	RemoveImage(ctx context.Context, path string) error
}

type imageUploadService struct{}

func NewImageUploadService() ImageUploadService {
	return &imageUploadService{}
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
