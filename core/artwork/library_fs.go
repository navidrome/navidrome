package artwork

import (
	"context"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/core/storage"
	"github.com/navidrome/navidrome/model"
)

// libraryView bundles the MusicFS for a library with its absolute root path,
// so readers can open library-relative paths through FS and compose absolute
// paths (for ffmpeg, which is path-based) via Abs.
type libraryView struct {
	FS      storage.MusicFS
	absRoot string
}

// Abs returns the absolute path for a library-relative path. Returns "" for an
// empty rel so callers (fromFFmpegTag) can treat it as "no path available".
func (v libraryView) Abs(rel string) string {
	if rel == "" {
		return ""
	}
	return filepath.Join(v.absRoot, rel)
}

// loadLibraryView resolves the MusicFS and absolute root path in a single
// library lookup.
func loadLibraryView(ctx context.Context, ds model.DataStore, libID int) (libraryView, error) {
	lib, err := ds.Library(ctx).Get(libID)
	if err != nil {
		return libraryView{}, err
	}
	s, err := storage.For(lib.Path)
	if err != nil {
		return libraryView{}, err
	}
	fs, err := s.FS()
	if err != nil {
		return libraryView{}, err
	}
	return libraryView{FS: fs, absRoot: localOSRoot(lib.Path)}, nil
}

// localOSRoot maps a library path to its on-disk root so Abs() yields paths os.Open/os.Stat accept:
// a file:// URL becomes its parsed OS path (bare paths already are; non-local schemes stay unchanged).
func localOSRoot(libPath string) string {
	if !strings.Contains(libPath, "://") {
		return libPath
	}
	u, err := url.Parse(libPath)
	if err != nil || u.Scheme != storage.LocalSchemaID {
		return libPath
	}
	// Windows drive URLs (file://C:/Music) put the volume in Host; rejoin it, matching
	// core/storage/local's newLocalStorage so os.Open/os.Stat get a valid path.
	if filepath.VolumeName(u.Host) != "" {
		return filepath.Join(u.Host, u.Path)
	}
	return u.Path
}
