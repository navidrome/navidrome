package artwork

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zeebo/xxh3"
)

func HashImage(r io.Reader) (string, error) {
	d := xxh3.New()
	if _, err := io.Copy(d, r); err != nil {
		return "", err
	}
	return fmt.Sprintf("%016x", d.Sum64()), nil
}

// ImageStore is the content-addressed store for artwork images that have no
// library file backing them (external downloads, embedded extractions, generated).
type ImageStore struct {
	root string
}

func NewImageStore(rootDir string) *ImageStore {
	return &ImageStore{root: rootDir}
}

// extForMime is deliberately NOT mime.ExtensionsByType: extensions are baked into
// content-addressed paths and re-derived on Open, so they must be stable across OSes.
func extForMime(m string) string {
	switch m {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	}
	return ".img"
}

func (s *ImageStore) path(hash, mimeType string) string {
	return filepath.Join(s.root, hash[0:2], hash[2:4], hash+extForMime(mimeType))
}

func (s *ImageStore) Write(hash, mimeType string, r io.Reader) error {
	dst := s.path(hash, mimeType)
	if _, err := os.Stat(dst); err == nil {
		// A touched mtime marks the file live so a concurrent prune spares it.
		now := time.Now()
		if err := os.Chtimes(dst, now, now); err == nil {
			return nil
		}
		// touch failed (file likely pruned concurrently) — fall through and write it
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dst), "."+hash+".tmp*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), dst)
}

func (s *ImageStore) Open(hash, mimeType string) (io.ReadCloser, error) {
	return os.Open(s.path(hash, mimeType))
}

// Remove deletes the store file unless it is newer than olderThan, in which case
// an overlapping acquisition may have just touched it and be about to commit its row.
func (s *ImageStore) Remove(hash, mimeType string, olderThan time.Time) error {
	path := s.path(hash, mimeType)
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.ModTime().After(olderThan) {
		return nil
	}
	err = os.Remove(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// Sweep removes store files not accepted by keep. Files modified after cutoff
// (including temp files) are always kept: their acquisition row may not be committed yet.
func (s *ImageStore) Sweep(cutoff time.Time, keep func(hash, ext string) bool) (int, error) {
	removed := 0
	err := filepath.WalkDir(s.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.ModTime().After(cutoff) {
			return nil
		}
		name := d.Name()
		remove := strings.HasPrefix(name, ".") // abandoned temp file past the grace window
		if !remove {
			ext := filepath.Ext(name)
			remove = !keep(strings.TrimSuffix(name, ext), ext)
		}
		if remove {
			// #nosec G122 -- path comes from WalkDir over our own store root, no attacker-controlled symlinks
			if err := os.Remove(path); err != nil {
				return err
			}
			removed++
		}
		return nil
	})
	if errors.Is(err, fs.ErrNotExist) {
		return removed, nil
	}
	return removed, err
}
