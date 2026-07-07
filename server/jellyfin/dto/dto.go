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
	Name     string `json:"Name"`
	ServerId string `json:"ServerId,omitempty"`
	Id       string `json:"Id"`
	// PlaylistItemId identifies an entry within a playlist listing (GET /Playlists/{id}/Items),
	// distinct from Id so a song appearing more than once can be removed by occurrence
	// (DELETE .../Items?EntryIds=...) rather than by song id.
	PlaylistItemId    string `json:"PlaylistItemId,omitempty"`
	Type              string `json:"Type"`
	IsFolder          bool   `json:"IsFolder"`
	MediaType         string `json:"MediaType,omitempty"`
	CollectionType    string `json:"CollectionType,omitempty"`
	LocationType      string `json:"LocationType,omitempty"`
	HasLyrics         bool   `json:"HasLyrics,omitempty"`
	SortName          string `json:"SortName,omitempty"`
	Path              string `json:"Path,omitempty"`
	ParentId          string `json:"ParentId,omitempty"`
	RunTimeTicks      int64  `json:"RunTimeTicks,omitempty"`
	IndexNumber       *int   `json:"IndexNumber,omitempty"`
	ParentIndexNumber *int   `json:"ParentIndexNumber,omitempty"`
	ProductionYear    *int   `json:"ProductionYear,omitempty"`
	// PremiereDate is the ISO 8601 release date; Finamp sorts "Latest Releases" by it client-side.
	PremiereDate *string `json:"PremiereDate,omitempty"`
	// DateCreated is the ISO 8601 date the item was added to the library; clients show it as
	// "Date Added" and sort "Recently Added" by it.
	DateCreated          string            `json:"DateCreated,omitempty"`
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
	// ImageBlurHashes is keyed by image type (e.g. "Primary") then image tag. Finamp uses it as a
	// de-dup key for image downloads (and a placeholder); absent, it warns the server isn't
	// calculating blurhashes.
	ImageBlurHashes   map[string]map[string]string `json:"ImageBlurHashes,omitempty"`
	BackdropImageTags []string                     `json:"BackdropImageTags"`
	UserData          *UserItemDataDto             `json:"UserData,omitempty"`
	MediaSources      []MediaSourceInfo            `json:"MediaSources,omitempty"`
	Container         string                       `json:"Container,omitempty"`
	CanDownload       bool                         `json:"CanDownload"`
}

// PlaylistUserPermissions is the response shape for GET /Playlists/{id}/Users(/{userId}), which
// Finamp probes before allowing playlist edits.
type PlaylistUserPermissions struct {
	UserId  string `json:"UserId"`
	CanEdit bool   `json:"CanEdit"`
}

// PlaylistInfo is the response shape for GET /Playlists/{id}. ItemIds are media item ids, not
// playlist-entry ids (matching real Jellyfin); Finamp reads OpenAccess for the public-visibility toggle.
type PlaylistInfo struct {
	OpenAccess bool                      `json:"OpenAccess"`
	Shares     []PlaylistUserPermissions `json:"Shares"`
	ItemIds    []string                  `json:"ItemIds"`
}

type QueryResult struct {
	Items            []BaseItemDto `json:"Items"`
	TotalRecordCount int           `json:"TotalRecordCount"`
	StartIndex       int           `json:"StartIndex"`
}

type UserDto struct {
	Name                      string             `json:"Name"`
	ServerId                  string             `json:"ServerId,omitempty"`
	ServerName                string             `json:"ServerName,omitempty"`
	Id                        string             `json:"Id"`
	HasPassword               bool               `json:"HasPassword"`
	HasConfiguredPassword     bool               `json:"HasConfiguredPassword"`
	HasConfiguredEasyPassword bool               `json:"HasConfiguredEasyPassword"`
	PrimaryImageTag           string             `json:"PrimaryImageTag,omitempty"`
	Policy                    *UserPolicy        `json:"Policy,omitempty"`
	Configuration             *UserConfiguration `json:"Configuration,omitempty"`
}

// UserPolicy mirrors real Jellyfin's User.Policy. Finamp reads it right after login and crashes if
// it's absent, so every field must be present even though Navidrome lacks most of these concepts.
type UserPolicy struct {
	IsAdministrator                  bool     `json:"IsAdministrator"`
	IsHidden                         bool     `json:"IsHidden"`
	EnableCollectionManagement       bool     `json:"EnableCollectionManagement"`
	EnableSubtitleManagement         bool     `json:"EnableSubtitleManagement"`
	EnableLyricManagement            bool     `json:"EnableLyricManagement"`
	IsDisabled                       bool     `json:"IsDisabled"`
	BlockedTags                      []string `json:"BlockedTags"`
	AllowedTags                      []string `json:"AllowedTags"`
	EnableUserPreferenceAccess       bool     `json:"EnableUserPreferenceAccess"`
	AccessSchedules                  []string `json:"AccessSchedules"`
	BlockUnratedItems                []string `json:"BlockUnratedItems"`
	EnableRemoteControlOfOtherUsers  bool     `json:"EnableRemoteControlOfOtherUsers"`
	EnableSharedDeviceControl        bool     `json:"EnableSharedDeviceControl"`
	EnableRemoteAccess               bool     `json:"EnableRemoteAccess"`
	EnableLiveTvManagement           bool     `json:"EnableLiveTvManagement"`
	EnableLiveTvAccess               bool     `json:"EnableLiveTvAccess"`
	EnableMediaPlayback              bool     `json:"EnableMediaPlayback"`
	EnableAudioPlaybackTranscoding   bool     `json:"EnableAudioPlaybackTranscoding"`
	EnableVideoPlaybackTranscoding   bool     `json:"EnableVideoPlaybackTranscoding"`
	EnablePlaybackRemuxing           bool     `json:"EnablePlaybackRemuxing"`
	ForceRemoteSourceTranscoding     bool     `json:"ForceRemoteSourceTranscoding"`
	EnableContentDeletion            bool     `json:"EnableContentDeletion"`
	EnableContentDeletionFromFolders []string `json:"EnableContentDeletionFromFolders"`
	EnableContentDownloading         bool     `json:"EnableContentDownloading"`
	EnableSyncTranscoding            bool     `json:"EnableSyncTranscoding"`
	EnableMediaConversion            bool     `json:"EnableMediaConversion"`
	EnabledDevices                   []string `json:"EnabledDevices"`
	EnableAllDevices                 bool     `json:"EnableAllDevices"`
	EnabledChannels                  []string `json:"EnabledChannels"`
	EnableAllChannels                bool     `json:"EnableAllChannels"`
	EnabledFolders                   []string `json:"EnabledFolders"`
	EnableAllFolders                 bool     `json:"EnableAllFolders"`
	InvalidLoginAttemptCount         int      `json:"InvalidLoginAttemptCount"`
	LoginAttemptsBeforeLockout       int      `json:"LoginAttemptsBeforeLockout"`
	MaxActiveSessions                int      `json:"MaxActiveSessions"`
	EnablePublicSharing              bool     `json:"EnablePublicSharing"`
	BlockedMediaFolders              []string `json:"BlockedMediaFolders"`
	BlockedChannels                  []string `json:"BlockedChannels"`
	RemoteClientBitrateLimit         int      `json:"RemoteClientBitrateLimit"`
	AuthenticationProviderId         string   `json:"AuthenticationProviderId"`
	PasswordResetProviderId          string   `json:"PasswordResetProviderId"`
	SyncPlayAccess                   string   `json:"SyncPlayAccess"`
}

// UserConfiguration mirrors real Jellyfin's User.Configuration. Like UserPolicy, clients expect it
// always present, even though most settings don't apply to Navidrome's audio-only library.
type UserConfiguration struct {
	PlayDefaultAudioTrack      bool     `json:"PlayDefaultAudioTrack"`
	SubtitleLanguagePreference string   `json:"SubtitleLanguagePreference"`
	DisplayMissingEpisodes     bool     `json:"DisplayMissingEpisodes"`
	GroupedFolders             []string `json:"GroupedFolders"`
	SubtitleMode               string   `json:"SubtitleMode"`
	DisplayCollectionsView     bool     `json:"DisplayCollectionsView"`
	EnableLocalPassword        bool     `json:"EnableLocalPassword"`
	OrderedViews               []string `json:"OrderedViews"`
	LatestItemsExcludes        []string `json:"LatestItemsExcludes"`
	MyMediaExcludes            []string `json:"MyMediaExcludes"`
	HidePlayedInLatest         bool     `json:"HidePlayedInLatest"`
	RememberAudioSelections    bool     `json:"RememberAudioSelections"`
	RememberSubtitleSelections bool     `json:"RememberSubtitleSelections"`
	EnableNextEpisodeAutoPlay  bool     `json:"EnableNextEpisodeAutoPlay"`
	CastReceiverId             string   `json:"CastReceiverId"`
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

// MediaStream mirrors real Jellyfin's MediaStream. Finamp declares several bools as non-nullable, so
// they must always be emitted (no omitempty). Finamp also does MediaStreams.firstWhere((s) => s.type
// == 'Audio'), so MediaSourceInfo must include at least one Audio stream or that lookup throws.
type MediaStream struct {
	Codec                  string `json:"Codec,omitempty"`
	Type                   string `json:"Type"`
	Index                  int    `json:"Index"`
	BitRate                int    `json:"BitRate,omitempty"`
	Channels               int    `json:"Channels,omitempty"`
	SampleRate             int    `json:"SampleRate,omitempty"`
	ChannelLayout          string `json:"ChannelLayout,omitempty"`
	IsInterlaced           bool   `json:"IsInterlaced"`
	IsDefault              bool   `json:"IsDefault"`
	IsForced               bool   `json:"IsForced"`
	IsExternal             bool   `json:"IsExternal"`
	IsTextSubtitleStream   bool   `json:"IsTextSubtitleStream"`
	SupportsExternalStream bool   `json:"SupportsExternalStream"`
}

// MediaSourceInfo mirrors real Jellyfin's MediaSourceInfo. Finamp declares several bools/arrays as
// non-nullable, so a missing field deserializes to null and throws a cast error that aborts parsing
// of the whole item list; emit them always (no omitempty on bools).
type MediaSourceInfo struct {
	Id                                  string        `json:"Id"`
	Path                                string        `json:"Path,omitempty"`
	Protocol                            string        `json:"Protocol"`
	Container                           string        `json:"Container,omitempty"`
	TranscodingUrl                      string        `json:"TranscodingUrl,omitempty"`
	TranscodingSubProtocol              string        `json:"TranscodingSubProtocol,omitempty"`
	Size                                int64         `json:"Size,omitempty"`
	Name                                string        `json:"Name,omitempty"`
	IsRemote                            bool          `json:"IsRemote"`
	RunTimeTicks                        int64         `json:"RunTimeTicks,omitempty"`
	Bitrate                             int           `json:"Bitrate,omitempty"`
	SupportsTranscoding                 bool          `json:"SupportsTranscoding"`
	SupportsDirectStream                bool          `json:"SupportsDirectStream"`
	SupportsDirectPlay                  bool          `json:"SupportsDirectPlay"`
	Type                                string        `json:"Type"`
	ReadAtNativeFramerate               bool          `json:"ReadAtNativeFramerate"`
	IgnoreDts                           bool          `json:"IgnoreDts"`
	IgnoreIndex                         bool          `json:"IgnoreIndex"`
	GenPtsInput                         bool          `json:"GenPtsInput"`
	IsInfiniteStream                    bool          `json:"IsInfiniteStream"`
	UseMostCompatibleTranscodingProfile bool          `json:"UseMostCompatibleTranscodingProfile"`
	RequiresOpening                     bool          `json:"RequiresOpening"`
	RequiresClosing                     bool          `json:"RequiresClosing"`
	RequiresLooping                     bool          `json:"RequiresLooping"`
	SupportsProbing                     bool          `json:"SupportsProbing"`
	HasSegments                         bool          `json:"HasSegments"`
	MediaStreams                        []MediaStream `json:"MediaStreams"`
	MediaAttachments                    []any         `json:"MediaAttachments"`
	Formats                             []string      `json:"Formats"`
}

type PlaybackInfoResponse struct {
	MediaSources  []MediaSourceInfo `json:"MediaSources"`
	PlaySessionId string            `json:"PlaySessionId"`
}
