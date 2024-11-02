package core

import (
	"context"
	"path/filepath"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

func userName(ctx context.Context) string {
	if user, ok := request.UserFrom(ctx); !ok {
		return "UNKNOWN"
	} else {
		return user.UserName
	}
}

// TODO We should only access files through the `storage.Storage` interface. This will require changing how
// TagLib and ffmpeg access files
func AbsolutePath(ctx context.Context, ds model.DataStore, libId int, path string) string {
	libPath, err := ds.Library(ctx).GetPath(libId)
	if err != nil {
		return path
	}
	return filepath.Join(libPath, path)
}
