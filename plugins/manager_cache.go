package plugins

import (
	"cmp"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/navidrome/navidrome/log"
)

// purgeCacheBySize removes the oldest files in dir until its total size is
// lower than or equal to maxSize. maxSize should be a human-readable string
// like "10MB" or "200K". If parsing fails or maxSize is "0", the function is
// a no-op.
func purgeCacheBySize(ctx context.Context, dir, maxSize string) {
	sizeLimit, err := humanize.ParseBytes(maxSize)
	if err != nil || sizeLimit == 0 {
		return
	}

	type fileInfo struct {
		path string
		size uint64
		mod  int64
	}

	var files []fileInfo
	var total uint64

	walk := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Trace(ctx, "Failed to access plugin cache entry", "path", path, err)
			return nil //nolint:nilerr
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			log.Trace(ctx, "Failed to get file info for plugin cache entry", "path", path, err)
			return nil //nolint:nilerr
		}
		files = append(files, fileInfo{
			path: path,
			size: uint64(info.Size()),
			mod:  info.ModTime().UnixMilli(),
		})
		total += uint64(info.Size())
		return nil
	}

	if err := filepath.WalkDir(dir, walk); err != nil {
		if !os.IsNotExist(err) {
			log.Warn(ctx, "Failed to traverse plugin cache directory", "path", dir, err)
		}
		return
	}

	log.Trace(ctx, "Current plugin cache size", "path", dir, "size", humanize.Bytes(total), "sizeLimit", humanize.Bytes(sizeLimit))
	if total <= sizeLimit {
		return
	}

	log.Debug(ctx, "Purging plugin cache", "path", dir, "sizeLimit", humanize.Bytes(sizeLimit), "currentSize", humanize.Bytes(total))
	slices.SortFunc(files, func(i, j fileInfo) int { return cmp.Compare(i.mod, j.mod) })

	for _, f := range files {
		if total <= sizeLimit {
			break
		}
		if err := os.Remove(f.path); err != nil {
			log.Warn(ctx, "Failed to remove plugin cache entry", "path", f.path, "size", humanize.Bytes(f.size), err)
			continue
		}
		total -= f.size
		log.Debug(ctx, "Removed plugin cache entry", "path", f.path, "size", humanize.Bytes(f.size), "time", time.UnixMilli(f.mod), "remainingSize", humanize.Bytes(total))

		// Remove empty parent directories
		dirPath := filepath.Dir(f.path)
		for dirPath != dir {
			if err := os.Remove(dirPath); err != nil {
				break
			}
			dirPath = filepath.Dir(dirPath)
		}
	}
}
