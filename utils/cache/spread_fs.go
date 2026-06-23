package cache

import (
	"crypto/sha1"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/djherbis/atime"
	"github.com/djherbis/fscache"
	"github.com/djherbis/stream"
	"github.com/navidrome/navidrome/log"
)

const completeMarkerSuffix = ".complete"
const sentinelName = ".complete-migrated"

type spreadFS struct {
	root string
	mode os.FileMode
	init func() error
}

// NewSpreadFS returns a FileSystem rooted at directory dir. This FS hashes the key and
// distributes all files in a layout like XX/XX/XXXXXXXXXX. Ex:
//
//		Key is abc123.300x300.jpg
//	    Hash would be: c574aeb3caafcf93ee337f0cf34e31a428ba3f13
//	    File in cache would be: c5 / 74 / c574aeb3caafcf93ee337f0cf34e31a428ba3f13
//
// The idea is to avoid having too many files in one dir, which could potentially cause performance issues
// and may hit limitations depending on the OS.
// See discussion here: https://github.com/djherbis/fscache/issues/8#issuecomment-614319323
//
// dir is created with specified mode if it doesn't exist.
func NewSpreadFS(dir string, mode os.FileMode) (*spreadFS, error) {
	f := &spreadFS{root: dir, mode: mode, init: func() error {
		return os.MkdirAll(dir, mode)
	}}
	return f, f.init()
}

func (sfs *spreadFS) Reload(f func(key string, name string)) error {
	sentinel := filepath.Join(sfs.root, sentinelName)
	migrating := false
	if _, err := os.Stat(sentinel); os.IsNotExist(err) {
		migrating = true
	}

	count := 0
	err := filepath.WalkDir(sfs.root, func(absoluteFilePath string, de fs.DirEntry, err error) error {
		if err != nil {
			log.Error("Error loading cache", "dir", sfs.root, err)
		}
		path, err := filepath.Rel(sfs.root, absoluteFilePath)
		if err != nil {
			return nil //nolint:nilerr
		}

		// Skip marker files; also clean orphan markers (data file gone).
		if strings.HasSuffix(path, completeMarkerSuffix) {
			dataPath := strings.TrimSuffix(absoluteFilePath, completeMarkerSuffix)
			if _, statErr := os.Stat(dataPath); os.IsNotExist(statErr) {
				_ = os.Remove(absoluteFilePath) //nolint:gosec
			}
			return nil
		}

		// Skip if name is not in the format XX/XX/XXXXXXXXXXXX
		parts := strings.Split(path, string(os.PathSeparator))
		if len(parts) != 3 || len(parts[0]) != 2 || len(parts[1]) != 2 || len(parts[2]) != 40 {
			return nil
		}

		_, markerErr := os.Stat(sfs.markerPath(absoluteFilePath))
		marked := markerErr == nil

		if !marked {
			if migrating {
				// First run after upgrade: trust existing files and mark them,
				// so they survive subsequent strict reloads.
				if mErr := sfs.MarkComplete(absoluteFilePath); mErr != nil {
					log.Warn("Error grandfathering cache file", "file", absoluteFilePath, mErr)
				}
			} else {
				// Steady state: an unmarked file is a crash partial. Discard it.
				log.Debug("Removing incomplete cache file", "file", absoluteFilePath)
				_ = os.Remove(absoluteFilePath) //nolint:gosec
				return nil
			}
		}

		f(absoluteFilePath, absoluteFilePath)
		count++
		return nil
	})

	if migrating {
		if sErr := os.WriteFile(sentinel, nil, 0600); sErr != nil {
			log.Warn("Error writing cache migration sentinel", "file", sentinel, sErr)
		}
	}
	if err == nil {
		log.Debug("Loaded cache", "dir", sfs.root, "numItems", count)
	}
	return err
}

func (sfs *spreadFS) Create(name string) (stream.File, error) {
	path := filepath.Dir(name)
	err := os.MkdirAll(path, sfs.mode)
	if err != nil {
		return nil, err
	}
	return os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
}

func (sfs *spreadFS) Open(name string) (stream.File, error) {
	return os.Open(name)
}

func (sfs *spreadFS) markerPath(dataPath string) string {
	return dataPath + completeMarkerSuffix
}

// MarkComplete records that the data file at dataPath was written in full.
// Only files with a marker are adopted on the next Reload; this is what
// distinguishes a complete cache entry from a partial one left by a crash.
func (sfs *spreadFS) MarkComplete(dataPath string) error {
	f, err := os.OpenFile(sfs.markerPath(dataPath), os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	return f.Close()
}

func (sfs *spreadFS) Remove(name string) error {
	if err := os.Remove(sfs.markerPath(name)); err != nil && !os.IsNotExist(err) {
		log.Warn("Error removing cache completion marker", "file", name, err)
	}
	return os.Remove(name)
}

func (sfs *spreadFS) Stat(name string) (fscache.FileInfo, error) {
	stat, err := os.Stat(name)
	if err != nil {
		return fscache.FileInfo{}, err
	}
	return fscache.FileInfo{FileInfo: stat, Atime: atime.Get(stat)}, nil
}

func (sfs *spreadFS) RemoveAll() error {
	if err := os.RemoveAll(sfs.root); err != nil {
		return err
	}
	return sfs.init()
}

func (sfs *spreadFS) KeyMapper(key string) string {
	// When running the Haunter, fscache can call this KeyMapper with the cached filepath instead of the key.
	// That's because we don't inform the original cache keys when reloading in the Reload function above.
	// If that's the case, just return the file path, as it is the actual mapped key.
	if strings.HasPrefix(key, sfs.root) {
		return key
	}
	hash := fmt.Sprintf("%x", sha1.Sum([]byte(key)))
	return filepath.Join(sfs.root, hash[0:2], hash[2:4], hash)
}
