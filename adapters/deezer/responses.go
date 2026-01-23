package deezer

type SearchArtistResults struct {
	Data  []Artist `json:"data"`
	Total int      `json:"total"`
	Next  string   `json:"next"`
}

type Artist struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Link          string `json:"link"`
	Picture       string `json:"picture"`
	PictureSmall  string `json:"picture_small"`
	PictureMedium string `json:"picture_medium"`
	PictureBig    string `json:"picture_big"`
	PictureXl     string `json:"picture_xl"`
	NbAlbum       int    `json:"nb_album"`
	NbFan         int    `json:"nb_fan"`
	Radio         bool   `json:"radio"`
	Tracklist     string `json:"tracklist"`
	Type          string `json:"type"`
}

type Error struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error"`
}

type RelatedArtists struct {
	Data  []Artist `json:"data"`
	Total int      `json:"total"`
}

type TopTracks struct {
	Data  []Track `json:"data"`
	Total int     `json:"total"`
	Next  string  `json:"next"`
}

type Track struct {
	ID           int      `json:"id"`
	Title        string   `json:"title"`
	Link         string   `json:"link"`
	Duration     int      `json:"duration"`
	Rank         int      `json:"rank"`
	Preview      string   `json:"preview"`
	Artist       Artist   `json:"artist"`
	Album        Album    `json:"album"`
	Contributors []Artist `json:"contributors"`
}

type Album struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Cover       string `json:"cover"`
	CoverSmall  string `json:"cover_small"`
	CoverMedium string `json:"cover_medium"`
	CoverBig    string `json:"cover_big"`
	CoverXl     string `json:"cover_xl"`
	Tracklist   string `json:"tracklist"`
	Type        string `json:"type"`
}
