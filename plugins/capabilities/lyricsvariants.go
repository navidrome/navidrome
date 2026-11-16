package capabilities

// LyricsVariants provides optional playback identities and exact source-native
// documents. It is additive to the required lyrics capability: plugins that do
// not implement it keep the classic GetLyrics behaviour.
//
//nd:capability name=lyricsvariants
type LyricsVariants interface {
	// Playback v2 returns the same playback entries as GetLyrics plus a
	// versioned identity and every cached representation from each source.
	//nd:export name=nd_lyrics_get_lyrics_v2
	GetLyricsV2(GetLyricsV2Request) (GetLyricsV2Response, error)
	// List returns the currently cached, addressable representations.
	//nd:export name=nd_lyrics_get_variants
	GetLyricsVariants(GetLyricsVariantsRequest) (GetLyricsVariantsResponse, error)
	// Original returns one exact native document for one selection.
	//nd:export name=nd_lyrics_get_original
	GetLyricsOriginal(GetLyricsOriginalRequest) (GetLyricsOriginalResponse, error)
}

type GetLyricsV2Request struct {
	Track TrackInfo `json:"track"`
}

type GetLyricsV2Response struct {
	Lyrics []LyricsV2Text `json:"lyrics"`
	Source *LyricsSource  `json:"source,omitempty"`
}

// LyricsV2Text is one playback entry with its optional addressable identity.
type LyricsV2Text struct {
	Lang    string         `json:"lang,omitempty"`
	Text    string         `json:"text"`
	Source  *LyricsSource  `json:"source,omitempty"`
	Variant *LyricsVariant `json:"variant,omitempty"`
}

// LyricsVariant is the versioned identity of one playback representation.
type LyricsVariant struct {
	Version   int    `json:"version"`
	Instance  string `json:"instance,omitempty"`
	Recording string `json:"recording,omitempty"`
	Language  string `json:"language,omitempty"`
	Role      string `json:"role,omitempty"`
	Format    string `json:"format,omitempty"`
}

type GetLyricsVariantsRequest struct {
	Track TrackInfo `json:"track"`
}

type GetLyricsVariantsResponse struct {
	Variants []LyricsVariantEntry `json:"variants"`
}

// LyricsVariantEntry is one listing row. PlaybackIndex is advisory: the
// identity is the stable link to the playback entry.
type LyricsVariantEntry struct {
	Identity      LyricsVariant      `json:"identity"`
	Provider      string             `json:"provider,omitempty"`
	Language      string             `json:"language,omitempty"`
	Role          string             `json:"role,omitempty"`
	Format        string             `json:"format,omitempty"`
	PlaybackIndex int                `json:"playbackIndex"`
	Original      LyricsOriginalInfo `json:"original"`
}

// LyricsOriginalInfo reports whether an exact native document is retrievable.
type LyricsOriginalInfo struct {
	Available  bool   `json:"available"`
	Format     string `json:"format,omitempty"`
	Provenance string `json:"provenance,omitempty"`
	MediaType  string `json:"mediaType,omitempty"`
	Extension  string `json:"extension,omitempty"`
	Size       int    `json:"size,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

type GetLyricsOriginalRequest struct {
	Track    TrackInfo     `json:"track"`
	Identity LyricsVariant `json:"identity"`
}

type GetLyricsOriginalResponse struct {
	Identity          LyricsVariant `json:"identity"`
	Available         bool          `json:"available"`
	Format            string        `json:"format,omitempty"`
	Provenance        string        `json:"provenance,omitempty"`
	MediaType         string        `json:"mediaType,omitempty"`
	Extension         string        `json:"extension,omitempty"`
	Document          []byte        `json:"document,omitempty"`
	UnavailableReason string        `json:"unavailableReason,omitempty"`
}
