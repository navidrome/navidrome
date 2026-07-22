package artwork

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

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
		return nil
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

func (s *ImageStore) Remove(hash, mimeType string) error {
	err := os.Remove(s.path(hash, mimeType))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

func (s *ImageStore) Sweep(keep func(hash string) bool) (int, error) {
	removed := 0
	err := filepath.WalkDir(s.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		name := d.Name()
		if strings.HasPrefix(name, ".") { // in-flight temp files
			return nil
		}
		hash := strings.TrimSuffix(name, filepath.Ext(name))
		if !keep(hash) {
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
