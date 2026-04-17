package artwork

import (
	"context"

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
