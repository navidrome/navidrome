package lastfm

type Response struct {
	Artist Artist `json:"artist"`
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
	Similar struct {
		Artists []Artist `json:"artist"`
	} `json:"similar"`
	Tags struct {
		Tag []ArtistTag `json:"tag"`
	} `json:"tags"`
	Bio ArtistBio `json:"bio"`
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

type Error struct {
	Code    int    `json:"error"`
	Message string `json:"message"`
}
