package tidal

// SearchResponse represents the JSON:API response from Tidal search endpoint
type SearchResponse struct {
	Artists []ArtistResource `json:"data"`
}

// ArtistResource represents an artist in Tidal's JSON:API format
type ArtistResource struct {
	ID         string           `json:"id"`
	Type       string           `json:"type"`
	Attributes ArtistAttributes `json:"attributes"`
}

// ArtistAttributes contains the artist's metadata
type ArtistAttributes struct {
	Name       string   `json:"name"`
	Popularity int      `json:"popularity"`
	Picture    []Image  `json:"picture"`
	ExternalLinks []Link `json:"externalLinks,omitempty"`
}

// Image represents an image resource from Tidal
type Image struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// Link represents an external link
type Link struct {
	Href string `json:"href"`
	Meta struct {
		Type string `json:"type"`
	} `json:"meta,omitempty"`
}

// TracksResponse represents the response from artist top tracks endpoint
type TracksResponse struct {
	Data []TrackResource `json:"data"`
}

// TrackResource represents a track in Tidal's JSON:API format
type TrackResource struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Attributes TrackAttributes `json:"attributes"`
}

// TrackAttributes contains track metadata
type TrackAttributes struct {
	Title      string `json:"title"`
	ISRC       string `json:"isrc"`
	Duration   int    `json:"duration"` // Duration in seconds
	Popularity int    `json:"popularity"`
}

// SimilarArtistsResponse represents the response from similar artists endpoint
type SimilarArtistsResponse struct {
	Data []ArtistResource `json:"data"`
}

// AlbumsResponse represents the response from albums endpoint
type AlbumsResponse struct {
	Data []AlbumResource `json:"data"`
}

// AlbumResource represents an album in Tidal's JSON:API format
type AlbumResource struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Attributes AlbumAttributes `json:"attributes"`
}

// AlbumAttributes contains album metadata
type AlbumAttributes struct {
	Title       string  `json:"title"`
	ReleaseDate string  `json:"releaseDate"`
	Cover       []Image `json:"cover"`
}

// TokenResponse represents the OAuth token response
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// ErrorResponse represents an error from the Tidal API
type ErrorResponse struct {
	Errors []APIError `json:"errors"`
}

// APIError represents a single error in the errors array
type APIError struct {
	ID     string `json:"id"`
	Status int    `json:"status"`
	Code   string `json:"code"`
	Detail string `json:"detail"`
}
