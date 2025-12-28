package plugins

import (
	"context"
	"fmt"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/plugins/host"
)

type libraryServiceImpl struct {
	ds                model.DataStore
	hasFilesystemPerm bool
}

func newLibraryService(ds model.DataStore, perm *LibraryPermission) host.LibraryService {
	hasFS := perm != nil && perm.Filesystem
	return &libraryServiceImpl{
		ds:                ds,
		hasFilesystemPerm: hasFS,
	}
}

func (s *libraryServiceImpl) GetLibrary(ctx context.Context, id int32) (*host.Library, error) {
	lib, err := s.ds.Library(ctx).Get(int(id))
	if err != nil {
		return nil, fmt.Errorf("library not found: %w", err)
	}

	return s.convertLibrary(lib), nil
}

func (s *libraryServiceImpl) GetAllLibraries(ctx context.Context) ([]host.Library, error) {
	libs, err := s.ds.Library(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get libraries: %w", err)
	}

	result := make([]host.Library, len(libs))
	for i, lib := range libs {
		result[i] = *s.convertLibrary(&lib)
	}

	return result, nil
}

func (s *libraryServiceImpl) convertLibrary(lib *model.Library) *host.Library {
	hostLib := &host.Library{
		ID:            int32(lib.ID),
		Name:          lib.Name,
		LastScanAt:    lib.LastScanAt.Unix(),
		TotalSongs:    int32(lib.TotalSongs),
		TotalAlbums:   int32(lib.TotalAlbums),
		TotalArtists:  int32(lib.TotalArtists),
		TotalSize:     lib.TotalSize,
		TotalDuration: lib.TotalDuration,
	}

	// Only include path and mount point if filesystem permission is granted
	if s.hasFilesystemPerm {
		hostLib.Path = lib.Path
		hostLib.MountPoint = toPluginMountPoint(int32(lib.ID))
	}

	return hostLib
}

func toPluginMountPoint(libID int32) string {
	return fmt.Sprintf("/libraries/%d", libID)
}

var _ host.LibraryService = (*libraryServiceImpl)(nil)
