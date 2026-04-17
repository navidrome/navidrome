package artwork

import (
	"context"

	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/storage"
	"github.com/navidrome/navidrome/model"
)

// libraryFS resolves the storage MusicFS for the given library ID.
// Per-call lookup; readers should call this once at construction time.
func libraryFS(ctx context.Context, ds model.DataStore, libID int) (storage.MusicFS, error) {
	lib, err := ds.Library(ctx).Get(libID)
	if err != nil {
		return nil, err
	}
	s, err := storage.For(lib.Path)
	if err != nil {
		return nil, err
	}
	return s.FS()
}

// libraryFSAndRoot resolves the MusicFS and the library's root path for
// readers that need both. The returned root is the same value core.AbsolutePath
// produces for this library — readers use it to compose absolute paths for
// ffmpeg and to derive libFS-relative paths from absolute mediafile paths
// (both sources share the same derivation, so any path-cleaning applied
// here is applied symmetrically elsewhere).
func libraryFSAndRoot(ctx context.Context, ds model.DataStore, libID int) (storage.MusicFS, string, error) {
	fs, err := libraryFS(ctx, ds, libID)
	if err != nil {
		return nil, "", err
	}
	return fs, core.AbsolutePath(ctx, ds, libID, ""), nil
}
