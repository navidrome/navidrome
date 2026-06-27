package capabilities

// Lyrics provides lyrics for a given track from external sources.
//
//nd:capability name=lyrics required=true
type Lyrics interface {
	//nd:export name=nd_lyrics_get_lyrics
	GetLyrics(GetLyricsRequest) (GetLyricsResponse, error)
}

// GetLyricsRequest contains the track information for lyrics lookup.
type GetLyricsRequest struct {
	Track TrackInfo `json:"track"`
}

// GetLyricsResponse contains the lyrics returned by the plugin.
type GetLyricsResponse struct {
	Lyrics []LyricsText `json:"lyrics"`
}

// LyricsText represents a single set of lyrics in raw text format.
// Text can be plain text or LRC format — Navidrome will parse it.
type LyricsText struct {
	Lang string `json:"lang,omitempty"`
	Text string `json:"text"`
}
