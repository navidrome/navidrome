package host

import "context"

// Library represents a music library with metadata.
type Library struct {
	ID            int32   `json:"id"`
	Name          string  `json:"name"`
	Path          string  `json:"path,omitempty"`
	MountPoint    string  `json:"mountPoint,omitempty"`
	LastScanAt    int64   `json:"lastScanAt"`
	TotalSongs    int32   `json:"totalSongs"`
	TotalAlbums   int32   `json:"totalAlbums"`
	TotalArtists  int32   `json:"totalArtists"`
	TotalSize     int64   `json:"totalSize"`
	TotalDuration float64 `json:"totalDuration"`
}

// LibraryService provides access to music library metadata for plugins.
//
// This service allows plugins to query information about configured music libraries,
// including statistics and optionally filesystem access to library directories.
// Filesystem access is controlled via the `filesystem` permission flag.
//
//nd:hostservice name=Library permission=library
type LibraryService interface {
	// GetLibrary retrieves metadata for a specific library by ID.
	//
	// Parameters:
	//   - id: The library's unique identifier
	//
	// Returns the library metadata, or an error if the library is not found.
	//nd:hostfunc
	GetLibrary(ctx context.Context, id int32) (*Library, error)

	// GetAllLibraries retrieves metadata for all configured libraries.
	//
	// Returns a slice of all libraries with their metadata.
	//nd:hostfunc
	GetAllLibraries(ctx context.Context) ([]Library, error)
}
