package lastfm

type Response struct {
	Artist         Artist         `json:"artist"`
	SimilarArtists SimilarArtists `json:"similarartists"`
	TopTracks      TopTracks      `json:"toptracks"`
}

type Artist struct {
	Name       string        `json:"name"`
	MBID       string        `json:"mbid"`
	URL        string        `json:"url"`
	Image      []ArtistImage `json:"image"`
	Streamable string        `json:"streamable"`
	Stats      struct {
		Listeners string `json:"listeners"`
		Plays     string `json:"plays"`
	} `json:"stats"`
	Similar SimilarArtists `json:"similar"`
	Tags    struct {
		Tag []ArtistTag `json:"tag"`
	} `json:"tags"`
	Bio ArtistBio `json:"bio"`
}

type SimilarArtists struct {
	Artists []Artist `json:"artist"`
}

type ArtistImage struct {
	URL  string `json:"#text"`
	Size string `json:"size"`
}

type ArtistTag struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type ArtistBio struct {
	Published string `json:"published"`
	Summary   string `json:"summary"`
	Content   string `json:"content"`
}

type Track struct {
	Name string `json:"name"`
	MBID string `json:"mbid"`
}

type TopTracks struct {
	Track []Track `json:"track"`
}

type Error struct {
	Code    int    `json:"error"`
	Message string `json:"message"`
}
