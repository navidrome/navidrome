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
	GetArtistMBID(ArtistMBIDRequest) (*ArtistMBIDResponse, error)

	// GetArtistURL retrieves the external URL for an artist.
	//nd:export name=nd_get_artist_url
	GetArtistURL(ArtistRequest) (*ArtistURLResponse, error)

	// GetArtistBiography retrieves the biography for an artist.
	//nd:export name=nd_get_artist_biography
	GetArtistBiography(ArtistRequest) (*ArtistBiographyResponse, error)

	// GetSimilarArtists retrieves similar artists for a given artist.
	//nd:export name=nd_get_similar_artists
	GetSimilarArtists(SimilarArtistsRequest) (*SimilarArtistsResponse, error)

	// GetArtistImages retrieves images for an artist.
	//nd:export name=nd_get_artist_images
	GetArtistImages(ArtistRequest) (*ArtistImagesResponse, error)

	// GetArtistTopSongs retrieves top songs for an artist.
	//nd:export name=nd_get_artist_top_songs
	GetArtistTopSongs(TopSongsRequest) (*TopSongsResponse, error)

	// GetAlbumInfo retrieves album information.
	//nd:export name=nd_get_album_info
	GetAlbumInfo(AlbumRequest) (*AlbumInfoResponse, error)

	// GetAlbumImages retrieves images for an album.
	//nd:export name=nd_get_album_images
	GetAlbumImages(AlbumRequest) (*AlbumImagesResponse, error)

	// GetSimilarSongsByTrack retrieves songs similar to a specific track.
	//nd:export name=nd_get_similar_songs_by_track
	GetSimilarSongsByTrack(SimilarSongsByTrackRequest) (*SimilarSongsResponse, error)

	// GetSimilarSongsByAlbum retrieves songs similar to tracks on an album.
	//nd:export name=nd_get_similar_songs_by_album
	GetSimilarSongsByAlbum(SimilarSongsByAlbumRequest) (*SimilarSongsResponse, error)

	// GetSimilarSongsByArtist retrieves songs similar to an artist's catalog.
	//nd:export name=nd_get_similar_songs_by_artist
	GetSimilarSongsByArtist(SimilarSongsByArtistRequest) (*SimilarSongsResponse, error)
}

// ArtistMBIDRequest is the request for GetArtistMBID.
type ArtistMBIDRequest struct {
	// ID is the internal Navidrome artist ID.
	ID string `json:"id"`
	// Name is the artist name.
	Name string `json:"name"`
}

// ArtistMBIDResponse is the response for GetArtistMBID.
type ArtistMBIDResponse struct {
	// MBID is the MusicBrainz ID for the artist.
	MBID string `json:"mbid"`
}

// ArtistRequest is the common request for artist-related functions.
type ArtistRequest struct {
	// ID is the internal Navidrome artist ID.
	ID string `json:"id"`
	// Name is the artist name.
	Name string `json:"name"`
	// MBID is the MusicBrainz ID for the artist (if known).
	MBID string `json:"mbid,omitempty"`
}

// ArtistURLResponse is the response for GetArtistURL.
type ArtistURLResponse struct {
	// URL is the external URL for the artist.
	URL string `json:"url"`
}

// ArtistBiographyResponse is the response for GetArtistBiography.
type ArtistBiographyResponse struct {
	// Biography is the artist biography text.
	Biography string `json:"biography"`
}

// SimilarArtistsRequest is the request for GetSimilarArtists.
type SimilarArtistsRequest struct {
	// ID is the internal Navidrome artist ID.
	ID string `json:"id"`
	// Name is the artist name.
	Name string `json:"name"`
	// MBID is the MusicBrainz ID for the artist (if known).
	MBID string `json:"mbid,omitempty"`
	// Limit is the maximum number of similar artists to return.
	Limit int32 `json:"limit"`
}

// SimilarArtistsResponse is the response for GetSimilarArtists.
type SimilarArtistsResponse struct {
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

// ArtistImagesResponse is the response for GetArtistImages.
type ArtistImagesResponse struct {
	// Images is the list of artist images.
	Images []ImageInfo `json:"images"`
}

// TopSongsRequest is the request for GetArtistTopSongs.
type TopSongsRequest struct {
	// ID is the internal Navidrome artist ID.
	ID string `json:"id"`
	// Name is the artist name.
	Name string `json:"name"`
	// MBID is the MusicBrainz ID for the artist (if known).
	MBID string `json:"mbid,omitempty"`
	// Count is the maximum number of top songs to return.
	Count int32 `json:"count"`
}

// SongRef is a reference to a song with metadata for matching.
type SongRef struct {
	// ID is the internal Navidrome mediafile ID (if known).
	ID string `json:"id,omitempty"`
	// Name is the song name.
	Name string `json:"name"`
	// MBID is the MusicBrainz ID for the song.
	MBID string `json:"mbid,omitempty"`
	// Artist is the artist name.
	Artist string `json:"artist,omitempty"`
	// ArtistMBID is the MusicBrainz artist ID.
	ArtistMBID string `json:"artistMbid,omitempty"`
	// Album is the album name.
	Album string `json:"album,omitempty"`
	// AlbumMBID is the MusicBrainz release ID.
	AlbumMBID string `json:"albumMbid,omitempty"`
	// Duration is the song duration in seconds.
	Duration float32 `json:"duration,omitempty"`
}

// TopSongsResponse is the response for GetArtistTopSongs.
type TopSongsResponse struct {
	// Songs is the list of top songs.
	Songs []SongRef `json:"songs"`
}

// AlbumRequest is the common request for album-related functions.
type AlbumRequest struct {
	// Name is the album name.
	Name string `json:"name"`
	// Artist is the album artist name.
	Artist string `json:"artist"`
	// MBID is the MusicBrainz ID for the album (if known).
	MBID string `json:"mbid,omitempty"`
}

// AlbumInfoResponse is the response for GetAlbumInfo.
type AlbumInfoResponse struct {
	// Name is the album name.
	Name string `json:"name"`
	// MBID is the MusicBrainz ID for the album.
	MBID string `json:"mbid"`
	// Description is the album description/notes.
	Description string `json:"description"`
	// URL is the external URL for the album.
	URL string `json:"url"`
}

// AlbumImagesResponse is the response for GetAlbumImages.
type AlbumImagesResponse struct {
	// Images is the list of album images.
	Images []ImageInfo `json:"images"`
}

// SimilarSongsByTrackRequest is the request for GetSimilarSongsByTrack.
type SimilarSongsByTrackRequest struct {
	// ID is the internal Navidrome mediafile ID.
	ID string `json:"id"`
	// Name is the track title.
	Name string `json:"name"`
	// Artist is the artist name.
	Artist string `json:"artist"`
	// MBID is the MusicBrainz recording ID (if known).
	MBID string `json:"mbid,omitempty"`
	// Count is the maximum number of similar songs to return.
	Count int32 `json:"count"`
}

// SimilarSongsByAlbumRequest is the request for GetSimilarSongsByAlbum.
type SimilarSongsByAlbumRequest struct {
	// ID is the internal Navidrome album ID.
	ID string `json:"id"`
	// Name is the album name.
	Name string `json:"name"`
	// Artist is the album artist name.
	Artist string `json:"artist"`
	// MBID is the MusicBrainz release ID (if known).
	MBID string `json:"mbid,omitempty"`
	// Count is the maximum number of similar songs to return.
	Count int32 `json:"count"`
}

// SimilarSongsByArtistRequest is the request for GetSimilarSongsByArtist.
type SimilarSongsByArtistRequest struct {
	// ID is the internal Navidrome artist ID.
	ID string `json:"id"`
	// Name is the artist name.
	Name string `json:"name"`
	// MBID is the MusicBrainz artist ID (if known).
	MBID string `json:"mbid,omitempty"`
	// Count is the maximum number of similar songs to return.
	Count int32 `json:"count"`
}

// SimilarSongsResponse is the response for GetSimilarSongsBy* functions.
type SimilarSongsResponse struct {
	// Songs is the list of similar songs.
	Songs []SongRef `json:"songs"`
}
