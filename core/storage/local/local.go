package local

import (
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/djherbis/times"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/storage"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/metadata"
)

// localStorage implements a Storage that reads the files from the local filesystem and uses registered extractors
// to extract the metadata and tags from the files.
type localStorage struct {
	u            url.URL
	extractor    Extractor
	resolvedPath string
	watching     atomic.Bool
}

func newLocalStorage(u url.URL) storage.Storage {
	newExtractor, ok := extractors[conf.Server.Scanner.Extractor]
	if !ok || newExtractor == nil {
		if conf.Server.Scanner.Extractor != consts.DefaultScannerExtractor {
			log.Warn("Extractor not found, using default", "extractor", conf.Server.Scanner.Extractor, "default", consts.DefaultScannerExtractor)
		}
		newExtractor = extractors[consts.DefaultScannerExtractor]
		if newExtractor == nil {
			log.Fatal("Default extractor not registered", "extractor", consts.DefaultScannerExtractor)
		}
	}
	isWindowsPath := filepath.VolumeName(u.Host) != ""
	if u.Scheme == storage.LocalSchemaID && isWindowsPath {
		u.Path = filepath.Join(u.Host, u.Path)
	}
	resolvedPath, err := filepath.EvalSymlinks(u.Path)
	if err != nil {
		log.Warn("Error resolving path", "path", u.Path, "err", err)
		resolvedPath = u.Path
	}
	return &localStorage{u: u, extractor: newExtractor(os.DirFS(u.Path), u.Path), resolvedPath: resolvedPath}
}

func (s *localStorage) FS() (storage.MusicFS, error) {
	path := s.u.Path
	if _, err := os.Stat(path); err != nil { //nolint:gosec
		return nil, fmt.Errorf("%w: %s", err, path)
	}
	return &localFS{FS: os.DirFS(path), extractor: s.extractor, root: path}, nil
}

type localFS struct {
	fs.FS
	extractor Extractor
	root      string
	// devices whose statx never reports a birth time (NFS, rclone/FUSE), so we ask each only once
	noBirthTime sync.Map
}

// ResolveSymlink implements storage.SymlinkResolverFS. It resolves the whole chain at the
// OS level, so links whose targets live outside the library folder (not reachable through
// the fs.FS abstraction) still resolve to their final target.
func (lfs *localFS) ResolveSymlink(name string) (string, error) {
	if !fs.ValidPath(name) {
		return "", &fs.PathError{Op: "resolvesymlink", Path: name, Err: fs.ErrInvalid}
	}
	return filepath.EvalSymlinks(filepath.Join(lfs.root, filepath.FromSlash(name)))
}

func (lfs *localFS) ReadTags(path ...string) (map[string]metadata.Info, error) {
	res, err := lfs.extractor.Parse(path...)
	if err != nil {
		return nil, err
	}
	for path, v := range res {
		if v.FileInfo == nil {
			info, err := fs.Stat(lfs, path)
			if err != nil {
				return nil, err
			}
			v.FileInfo = localFileInfo{
				FileInfo:    info,
				path:        filepath.Join(lfs.root, filepath.FromSlash(path)),
				noBirthTime: &lfs.noBirthTime,
			}
			res[path] = v
		}
	}
	return res, nil
}

// localFileInfo is a wrapper around fs.FileInfo that adds a BirthTime method, to make it compatible
// with metadata.FileInfo
type localFileInfo struct {
	fs.FileInfo
	path        string
	noBirthTime *sync.Map
}

func (lfi localFileInfo) BirthTime() time.Time {
	if ts := times.Get(lfi.FileInfo); ts.HasBirthTime() {
		return ts.BirthTime()
	}
	if bt, ok := lfi.statxBirthTime(); ok {
		return bt
	}
	return time.Now()
}

// statxBirthTime reads the birth time from the path, which on Linux is the only way to get it.
// Filesystems that never report one are remembered per device, so a scan asks each only once.
func (lfi localFileInfo) statxBirthTime() (time.Time, bool) {
	if lfi.path == "" {
		return time.Time{}, false
	}
	dev, hasDev := deviceID(lfi.FileInfo)
	memo := lfi.noBirthTime
	if hasDev && memo != nil {
		if _, skip := memo.Load(dev); skip {
			return time.Time{}, false
		}
	}
	ts, err := times.Stat(lfi.path)
	if err != nil {
		return time.Time{}, false
	}
	if ts.HasBirthTime() {
		return ts.BirthTime(), true
	}
	if hasDev && memo != nil {
		memo.Store(dev, struct{}{})
	}
	return time.Time{}, false
}

func init() {
	storage.Register(storage.LocalSchemaID, newLocalStorage)
}
