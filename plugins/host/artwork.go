package host

import "context"

// ArtworkService provides artwork public URL generation capabilities for plugins.
//
// This service allows plugins to generate public URLs for artwork images of
// various entity types (artists, albums, tracks, playlists). The generated URLs
// include authentication tokens and can be used to display artwork in external
// services or custom UIs.
//
//nd:hostservice name=Artwork permission=artwork
type ArtworkService interface {
	// GetArtistUrl generates a public URL for an artist's artwork.
	//
	// Parameters:
	//   - id: The artist's unique identifier
	//   - size: Desired image size in pixels (0 for original size)
	//
	// Returns the public URL for the artwork, or an error if generation fails.
	//nd:hostfunc
	GetArtistUrl(ctx context.Context, id string, size int32) (url string, err error)

	// GetAlbumUrl generates a public URL for an album's artwork.
	//
	// Parameters:
	//   - id: The album's unique identifier
	//   - size: Desired image size in pixels (0 for original size)
	//
	// Returns the public URL for the artwork, or an error if generation fails.
	//nd:hostfunc
	GetAlbumUrl(ctx context.Context, id string, size int32) (url string, err error)

	// GetTrackUrl generates a public URL for a track's artwork.
	//
	// Parameters:
	//   - id: The track's (media file) unique identifier
	//   - size: Desired image size in pixels (0 for original size)
	//
	// Returns the public URL for the artwork, or an error if generation fails.
	//nd:hostfunc
	GetTrackUrl(ctx context.Context, id string, size int32) (url string, err error)

	// GetPlaylistUrl generates a public URL for a playlist's artwork.
	//
	// Parameters:
	//   - id: The playlist's unique identifier
	//   - size: Desired image size in pixels (0 for original size)
	//
	// Returns the public URL for the artwork, or an error if generation fails.
	//nd:hostfunc
	GetPlaylistUrl(ctx context.Context, id string, size int32) (url string, err error)
}
