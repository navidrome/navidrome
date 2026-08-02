package artwork

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/zeebo/xxh3"
)

// ImageStore is the content-addressed store for artwork images with no library file backing them.
type ImageStore struct {
	root string
}

func NewImageStore(rootDir string) *ImageStore {
	return &ImageStore{root: rootDir}
}

func GetImageStore() *ImageStore {
	return NewImageStore(filepath.Join(conf.Server.DataFolder.String(), consts.ArtworkFolder, consts.HashedArtworkFolder))
}

// extForMime must stay stable across OSes: extensions are baked into stored paths and re-derived on Open.
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

func hashImage(r io.Reader) (string, error) {
	d := xxh3.New()
	if _, err := io.Copy(d, r); err != nil {
		return "", err
	}
	return fmt.Sprintf("%016x", d.Sum64()), nil
}

// validHash guards path sharding: a malformed hash would slice-panic or inject path separators.
func validHash(hash string) bool {
	if len(hash) != 16 {
		return false
	}
	for _, c := range []byte(hash) {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

func (s *ImageStore) path(hash, mimeType string) string {
	return filepath.Join(s.root, hash[0:2], hash[2:4], hash+extForMime(mimeType))
}

func (s *ImageStore) Write(hash, mimeType string, r io.Reader) error {
	if !validHash(hash) {
		return fmt.Errorf("imagestore: invalid hash %q", hash)
	}
	dst := s.path(hash, mimeType)
	if _, err := os.Stat(dst); err == nil {
		// A touched mtime marks the file live so a concurrent prune spares it.
		now := time.Now()
		if err := os.Chtimes(dst, now, now); err == nil {
			return nil
		}
		// touch failed (likely pruned concurrently) — fall through and rewrite it
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
	if !validHash(hash) {
		return nil, fmt.Errorf("imagestore: invalid hash %q", hash)
	}
	return os.Open(s.path(hash, mimeType))
}

// Remove skips files newer than olderThan: an overlapping acquisition may not have committed its row yet.
func (s *ImageStore) Remove(hash, mimeType string, olderThan time.Time) error {
	if !validHash(hash) {
		return fmt.Errorf("imagestore: invalid hash %q", hash)
	}
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

// Sweep removes store files not accepted by keep. Files modified after cutoff are always
// kept: their acquisition row may not be committed yet.
func (s *ImageStore) Sweep(ctx context.Context, cutoff time.Time, keep func(hash, ext string) bool) (int, error) {
	removed := 0
	err := filepath.WalkDir(s.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if err := ctx.Err(); err != nil {
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
				// One unremovable file must not strand the rest of the store until the next prune.
				log.Warn(ctx, "Artwork: Could not remove store file", "path", path, err)
				return nil
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
