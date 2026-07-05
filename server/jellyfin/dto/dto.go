package dto

// PublicSystemInfo is the unauthenticated handshake payload (GET /System/Info/Public).
type PublicSystemInfo struct {
	LocalAddress           string `json:"LocalAddress,omitempty"`
	ServerName             string `json:"ServerName"`
	Version                string `json:"Version"`
	ProductName            string `json:"ProductName"`
	OperatingSystem        string `json:"OperatingSystem,omitempty"`
	Id                     string `json:"Id"`
	StartupWizardCompleted bool   `json:"StartupWizardCompleted"`
}

// SystemInfo is the authenticated variant (GET /System/Info).
type SystemInfo struct {
	PublicSystemInfo
	HasPendingRestart      bool   `json:"HasPendingRestart"`
	IsShuttingDown         bool   `json:"IsShuttingDown"`
	SupportsLibraryMonitor bool   `json:"SupportsLibraryMonitor"`
	CachePath              string `json:"CachePath,omitempty"`
}

type NameGuidPair struct {
	Name string `json:"Name"`
	Id   string `json:"Id"`
}

type UserItemDataDto struct {
	Rating                *float64 `json:"Rating,omitempty"`
	PlaybackPositionTicks int64    `json:"PlaybackPositionTicks"`
	PlayCount             int      `json:"PlayCount"`
	IsFavorite            bool     `json:"IsFavorite"`
	Played                bool     `json:"Played"`
	Key                   string   `json:"Key"`
	ItemId                string   `json:"ItemId,omitempty"`
	LastPlayedDate        *string  `json:"LastPlayedDate,omitempty"`
}

type BaseItemDto struct {
	Name                 string            `json:"Name"`
	ServerId             string            `json:"ServerId,omitempty"`
	Id                   string            `json:"Id"`
	Type                 string            `json:"Type"`
	IsFolder             bool              `json:"IsFolder"`
	MediaType            string            `json:"MediaType,omitempty"`
	CollectionType       string            `json:"CollectionType,omitempty"`
	ParentId             string            `json:"ParentId,omitempty"`
	RunTimeTicks         int64             `json:"RunTimeTicks,omitempty"`
	IndexNumber          *int              `json:"IndexNumber,omitempty"`
	ParentIndexNumber    *int              `json:"ParentIndexNumber,omitempty"`
	ProductionYear       *int              `json:"ProductionYear,omitempty"`
	Album                string            `json:"Album,omitempty"`
	AlbumId              string            `json:"AlbumId,omitempty"`
	AlbumArtist          string            `json:"AlbumArtist,omitempty"`
	AlbumArtists         []NameGuidPair    `json:"AlbumArtists,omitempty"`
	AlbumPrimaryImageTag string            `json:"AlbumPrimaryImageTag,omitempty"`
	Artists              []string          `json:"Artists,omitempty"`
	ArtistItems          []NameGuidPair    `json:"ArtistItems,omitempty"`
	Genres               []string          `json:"Genres,omitempty"`
	ChildCount           *int              `json:"ChildCount,omitempty"`
	SongCount            *int              `json:"SongCount,omitempty"`
	AlbumCount           *int              `json:"AlbumCount,omitempty"`
	ImageTags            map[string]string `json:"ImageTags,omitempty"`
	BackdropImageTags    []string          `json:"BackdropImageTags"`
	UserData             *UserItemDataDto  `json:"UserData,omitempty"`
	MediaSources         []MediaSourceInfo `json:"MediaSources,omitempty"`
	Container            string            `json:"Container,omitempty"`
	CanDownload          bool              `json:"CanDownload"`
}

type QueryResult struct {
	Items            []BaseItemDto `json:"Items"`
	TotalRecordCount int           `json:"TotalRecordCount"`
	StartIndex       int           `json:"StartIndex"`
}

type UserDto struct {
	Name                      string `json:"Name"`
	ServerId                  string `json:"ServerId,omitempty"`
	Id                        string `json:"Id"`
	HasPassword               bool   `json:"HasPassword"`
	HasConfiguredPassword     bool   `json:"HasConfiguredPassword"`
	HasConfiguredEasyPassword bool   `json:"HasConfiguredEasyPassword"`
	PrimaryImageTag           string `json:"PrimaryImageTag,omitempty"`
}

type SessionInfo struct {
	Id     string `json:"Id"`
	UserId string `json:"UserId"`
}

type AuthenticationResult struct {
	User        *UserDto     `json:"User"`
	SessionInfo *SessionInfo `json:"SessionInfo,omitempty"`
	AccessToken string       `json:"AccessToken"`
	ServerId    string       `json:"ServerId"`
}

type MediaSourceInfo struct {
	Id                   string `json:"Id"`
	Path                 string `json:"Path,omitempty"`
	Protocol             string `json:"Protocol"`
	Container            string `json:"Container,omitempty"`
	Size                 int64  `json:"Size,omitempty"`
	Name                 string `json:"Name,omitempty"`
	IsRemote             bool   `json:"IsRemote"`
	RunTimeTicks         int64  `json:"RunTimeTicks,omitempty"`
	SupportsTranscoding  bool   `json:"SupportsTranscoding"`
	SupportsDirectStream bool   `json:"SupportsDirectStream"`
	SupportsDirectPlay   bool   `json:"SupportsDirectPlay"`
	Type                 string `json:"Type"`
}

type PlaybackInfoResponse struct {
	MediaSources  []MediaSourceInfo `json:"MediaSources"`
	PlaySessionId string            `json:"PlaySessionId"`
}
