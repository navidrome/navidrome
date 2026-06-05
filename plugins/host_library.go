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
	allowedLibraryIDs []int
	allLibraries      bool
	libraryIDMap      map[int]struct{}
}

func newLibraryService(ds model.DataStore, perm *LibraryPermission, allowedLibraryIDs []int, allLibraries bool) host.LibraryService {
	hasFS := perm != nil && perm.Filesystem
	libraryIDMap := make(map[int]struct{})
	for _, id := range allowedLibraryIDs {
		libraryIDMap[id] = struct{}{}
	}
	return &libraryServiceImpl{
		ds:                ds,
		hasFilesystemPerm: hasFS,
		allowedLibraryIDs: allowedLibraryIDs,
		allLibraries:      allLibraries,
		libraryIDMap:      libraryIDMap,
	}
}

func (s *libraryServiceImpl) GetLibrary(ctx context.Context, id int32) (*host.Library, error) {
	// Check if the library is accessible
	if !s.isLibraryAccessible(int(id)) {
		return nil, fmt.Errorf("library not accessible: library ID %d is not in the allowed list", id)
	}

	lib, err := s.ds.Library(ctx).Get(int(id))
	if err != nil {
		return nil, fmt.Errorf("library not found: %w", err)
	}

	return s.convertLibrary(lib), nil
}

// isLibraryAccessible checks if a library ID is accessible to this plugin.
func (s *libraryServiceImpl) isLibraryAccessible(id int) bool {
	if s.allLibraries {
		return true
	}
	_, ok := s.libraryIDMap[id]
	return ok
}

func (s *libraryServiceImpl) GetAllLibraries(ctx context.Context) ([]host.Library, error) {
	libs, err := s.ds.Library(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get libraries: %w", err)
	}

	// Filter libraries based on allowed list
	var result []host.Library
	for _, lib := range libs {
		if s.isLibraryAccessible(lib.ID) {
			result = append(result, *s.convertLibrary(&lib))
		}
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
