package tagwriter

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
)

var (
	ErrFeatureDisabled = errors.New("tag editing is disabled in configuration")
	ErrUnsupportedFormat = errors.New("unsupported audio file format")
	ErrReadOnlyFile = errors.New("file is read-only at the OS level")
	ErrPermissionDenied = errors.New("permission denied")
)

type Tags map[string]string

const (
	TagTitle        = "title"
	TagArtist       = "artist"
	TagAlbum        = "album"
	TagAlbumArtist  = "albumartist"
	TagYear         = "year"
	TagGenre        = "genre"
	TagTrackNumber  = "tracknumber"
	TagTrackTotal   = "tracktotal"
	TagDiscNumber   = "discnumber"
	TagDiscTotal    = "disctotal"
	TagComment      = "comment"
	TagAlbumArt     = "albumart"
)

type TagWriter interface {
	WriteTags(filePath string, tags Tags) error
}

func New() TagWriter {
	return &tagWriter{}
}

type tagWriter struct{}

func (t *tagWriter) WriteTags(filePath string, tags Tags) error {
	if !conf.Server.EnableTagEditing {
		log.Debug("Tag editing is disabled. Enable with config option 'EnableTagEditing'")
		return ErrFeatureDisabled
	}

	if len(tags) == 0 {
		return nil
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	if err := t.checkFilePermissions(absPath); err != nil {
		return err
	}

	ext := strings.ToLower(filepath.Ext(absPath))

	lock, err := LockFile(absPath)
	if err != nil {
		return fmt.Errorf("failed to acquire file lock: %w", err)
	}
	defer func() {
		if unlockErr := UnlockFile(lock); unlockErr != nil {
			log.Error("Failed to release file lock", "filePath", absPath, "error", unlockErr)
		}
	}()

	var writeErr error
	switch ext {
	case ".mp3", ".mp2":
		writeErr = writeMP3Tags(absPath, tags)
	case ".flac":
		writeErr = writeFLACTags(absPath, tags)
	case ".wav", ".wave":
		writeErr = writeWAVTags(absPath, tags)
	case ".m4a", ".mp4":
		writeErr = writeM4ATags(absPath, tags)
	case ".ogg":
		writeErr = writeOGGTags(absPath, tags)
	default:
		return ErrUnsupportedFormat
	}

	if writeErr != nil {
		return fmt.Errorf("failed to write tags: %w", writeErr)
	}

	log.Debug("Tags written successfully", "filePath", absPath, "tags", tags)
	return nil
}

func (t *tagWriter) checkFilePermissions(filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %w", err)
		}
		return fmt.Errorf("failed to stat file: %w", err)
	}

	if info.Mode().IsDir() {
		return errors.New("path is a directory")
	}

	if info.Mode().Perm()&0200 == 0 {
		log.Warn("File is read-only, cannot write tags", "filePath", filePath)
		return ErrReadOnlyFile
	}

	return nil
}

func SupportedFormats() []string {
	return []string{".mp3", ".mp2", ".flac", ".wav", ".wave", ".m4a", ".mp4", ".ogg"}
}

func IsSupportedFormat(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	for _, supported := range SupportedFormats() {
		if ext == supported {
			return true
		}
	}
	return false
}