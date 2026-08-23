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
	Lyrics []LyricsText  `json:"lyrics"`
	Source *LyricsSource `json:"source,omitempty"`
}

// LyricsText represents a single set of lyrics in raw text format.
// Navidrome content-sniffs and parses the returned text.
type LyricsText struct {
	Lang string `json:"lang,omitempty"`
	Text string `json:"text"`
}

// LyricsSource identifies the optional upstream provider and format selected
// by a plugin. Navidrome supplies the plugin name and source type itself.
type LyricsSource struct {
	Provider string `json:"provider,omitempty"`
	Format   string `json:"format,omitempty"`
}
