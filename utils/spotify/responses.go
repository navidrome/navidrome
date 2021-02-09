package spotify

type SearchResults struct {
	Artists ArtistsResult `json:"artists"`
}

type ArtistsResult struct {
	HRef  string   `json:"href"`
	Items []Artist `json:"items"`
}

type Artist struct {
	Genres     []string `json:"genres"`
	HRef       string   `json:"href"`
	ID         string   `json:"id"`
	Popularity int      `json:"popularity"`
	Images     []Image  `json:"images"`
	Name       string   `json:"name"`
}

type Image struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type Error struct {
	Code    string `json:"error"`
	Message string `json:"error_description"`
}
