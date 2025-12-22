package plugins

// --- Input/Output JSON structures for MetadataAgent plugin calls ---

// artistMBIDInput is the input for GetArtistMBID
type artistMBIDInput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// artistMBIDOutput is the output for GetArtistMBID
type artistMBIDOutput struct {
	MBID string `json:"mbid"`
}

// artistInput is the common input for artist-related functions
type artistInput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	MBID string `json:"mbid,omitempty"`
}

// artistURLOutput is the output for GetArtistURL
type artistURLOutput struct {
	URL string `json:"url"`
}

// artistBiographyOutput is the output for GetArtistBiography
type artistBiographyOutput struct {
	Biography string `json:"biography"`
}

// similarArtistsInput is the input for GetSimilarArtists
type similarArtistsInput struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	MBID  string `json:"mbid,omitempty"`
	Limit int    `json:"limit"`
}

// artistRef is a reference to an artist with name and optional MBID
type artistRef struct {
	Name string `json:"name"`
	MBID string `json:"mbid,omitempty"`
}

// similarArtistsOutput is the output for GetSimilarArtists
type similarArtistsOutput struct {
	Artists []artistRef `json:"artists"`
}

// imageInfo represents an image with URL and size
type imageInfo struct {
	URL  string `json:"url"`
	Size int    `json:"size"`
}

// artistImagesOutput is the output for GetArtistImages
type artistImagesOutput struct {
	Images []imageInfo `json:"images"`
}

// topSongsInput is the input for GetArtistTopSongs
type topSongsInput struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	MBID  string `json:"mbid,omitempty"`
	Count int    `json:"count"`
}

// songRef is a reference to a song with name and optional MBID
type songRef struct {
	Name string `json:"name"`
	MBID string `json:"mbid,omitempty"`
}

// topSongsOutput is the output for GetArtistTopSongs
type topSongsOutput struct {
	Songs []songRef `json:"songs"`
}

// albumInput is the common input for album-related functions
type albumInput struct {
	Name   string `json:"name"`
	Artist string `json:"artist"`
	MBID   string `json:"mbid,omitempty"`
}

// albumInfoOutput is the output for GetAlbumInfo
type albumInfoOutput struct {
	Name        string `json:"name"`
	MBID        string `json:"mbid"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

// albumImagesOutput is the output for GetAlbumImages
type albumImagesOutput struct {
	Images []imageInfo `json:"images"`
}
