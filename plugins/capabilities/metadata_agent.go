package capabilities

// MetadataAgent provides artist and album metadata retrieval.
// This capability allows plugins to provide external metadata for artists and albums,
// such as biographies, images, similar artists, and top songs.
//
// Plugins implementing this capability can choose which methods to implement.
// Each method is optional - plugins only need to provide the functionality they support.
//
//nd:capability name=metadata
type MetadataAgent interface {
	// GetArtistMBID retrieves the MusicBrainz ID for an artist.
	//nd:export name=nd_get_artist_mbid
	GetArtistMBID(ArtistMBIDInput) (ArtistMBIDOutput, error)

	// GetArtistURL retrieves the external URL for an artist.
	//nd:export name=nd_get_artist_url
	GetArtistURL(ArtistInput) (ArtistURLOutput, error)

	// GetArtistBiography retrieves the biography for an artist.
	//nd:export name=nd_get_artist_biography
	GetArtistBiography(ArtistInput) (ArtistBiographyOutput, error)

	// GetSimilarArtists retrieves similar artists for a given artist.
	//nd:export name=nd_get_similar_artists
	GetSimilarArtists(SimilarArtistsInput) (SimilarArtistsOutput, error)

	// GetArtistImages retrieves images for an artist.
	//nd:export name=nd_get_artist_images
	GetArtistImages(ArtistInput) (ArtistImagesOutput, error)

	// GetArtistTopSongs retrieves top songs for an artist.
	//nd:export name=nd_get_artist_top_songs
	GetArtistTopSongs(TopSongsInput) (TopSongsOutput, error)

	// GetAlbumInfo retrieves album information.
	//nd:export name=nd_get_album_info
	GetAlbumInfo(AlbumInput) (AlbumInfoOutput, error)

	// GetAlbumImages retrieves images for an album.
	//nd:export name=nd_get_album_images
	GetAlbumImages(AlbumInput) (AlbumImagesOutput, error)
}

// ArtistMBIDInput is the input for GetArtistMBID.
type ArtistMBIDInput struct {
	// ID is the internal Navidrome artist ID.
	ID string `json:"id"`
	// Name is the artist name.
	Name string `json:"name"`
}

// ArtistMBIDOutput is the output for GetArtistMBID.
type ArtistMBIDOutput struct {
	// MBID is the MusicBrainz ID for the artist.
	MBID string `json:"mbid"`
}

// ArtistInput is the common input for artist-related functions.
type ArtistInput struct {
	// ID is the internal Navidrome artist ID.
	ID string `json:"id"`
	// Name is the artist name.
	Name string `json:"name"`
	// MBID is the MusicBrainz ID for the artist (if known).
	MBID *string `json:"mbid,omitempty"`
}

// ArtistURLOutput is the output for GetArtistURL.
type ArtistURLOutput struct {
	// URL is the external URL for the artist.
	URL string `json:"url"`
}

// ArtistBiographyOutput is the output for GetArtistBiography.
type ArtistBiographyOutput struct {
	// Biography is the artist biography text.
	Biography string `json:"biography"`
}

// SimilarArtistsInput is the input for GetSimilarArtists.
type SimilarArtistsInput struct {
	// ID is the internal Navidrome artist ID.
	ID string `json:"id"`
	// Name is the artist name.
	Name string `json:"name"`
	// MBID is the MusicBrainz ID for the artist (if known).
	MBID *string `json:"mbid,omitempty"`
	// Limit is the maximum number of similar artists to return.
	Limit int32 `json:"limit"`
}

// ArtistRef is a reference to an artist with name and optional MBID.
type ArtistRef struct {
	// Name is the artist name.
	Name string `json:"name"`
	// MBID is the MusicBrainz ID for the artist.
	MBID *string `json:"mbid,omitempty"`
}

// SimilarArtistsOutput is the output for GetSimilarArtists.
type SimilarArtistsOutput struct {
	// Artists is the list of similar artists.
	Artists []ArtistRef `json:"artists"`
}

// ImageInfo represents an image with URL and size.
type ImageInfo struct {
	// URL is the URL of the image.
	URL string `json:"url"`
	// Size is the size of the image in pixels (width or height).
	Size int32 `json:"size"`
}

// ArtistImagesOutput is the output for GetArtistImages.
type ArtistImagesOutput struct {
	// Images is the list of artist images.
	Images []ImageInfo `json:"images"`
}

// TopSongsInput is the input for GetArtistTopSongs.
type TopSongsInput struct {
	// ID is the internal Navidrome artist ID.
	ID string `json:"id"`
	// Name is the artist name.
	Name string `json:"name"`
	// MBID is the MusicBrainz ID for the artist (if known).
	MBID *string `json:"mbid,omitempty"`
	// Count is the maximum number of top songs to return.
	Count int32 `json:"count"`
}

// SongRef is a reference to a song with name and optional MBID.
type SongRef struct {
	// Name is the song name.
	Name string `json:"name"`
	// MBID is the MusicBrainz ID for the song.
	MBID *string `json:"mbid,omitempty"`
}

// TopSongsOutput is the output for GetArtistTopSongs.
type TopSongsOutput struct {
	// Songs is the list of top songs.
	Songs []SongRef `json:"songs"`
}

// AlbumInput is the common input for album-related functions.
type AlbumInput struct {
	// Name is the album name.
	Name string `json:"name"`
	// Artist is the album artist name.
	Artist string `json:"artist"`
	// MBID is the MusicBrainz ID for the album (if known).
	MBID *string `json:"mbid,omitempty"`
}

// AlbumInfoOutput is the output for GetAlbumInfo.
type AlbumInfoOutput struct {
	// Name is the album name.
	Name string `json:"name"`
	// MBID is the MusicBrainz ID for the album.
	MBID string `json:"mbid"`
	// Description is the album description/notes.
	Description string `json:"description"`
	// URL is the external URL for the album.
	URL string `json:"url"`
}

// AlbumImagesOutput is the output for GetAlbumImages.
type AlbumImagesOutput struct {
	// Images is the list of album images.
	Images []ImageInfo `json:"images"`
}
