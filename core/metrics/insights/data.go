package insights

import "time"

type Data struct {
	InsightsID string `json:"id"`
	Version    string `json:"version"`
	Uptime     int64  `json:"uptime"`
	Build      struct {
		Settings  map[string]string `json:"settings"`
		GoVersion string            `json:"goVersion"`
	} `json:"build"`
	OS struct {
		Type    string `json:"type"`
		Distro  string `json:"distro,omitempty"`
		Version string `json:"version,omitempty"`
		Arch    string `json:"arch"`
		NumCPU  int    `json:"numCPU"`
	} `json:"os"`
	FS struct {
		Music  *FSInfo `json:"music,omitempty"`
		Data   *FSInfo `json:"data,omitempty"`
		Cache  *FSInfo `json:"cache,omitempty"`
		Backup *FSInfo `json:"backup,omitempty"`
	}
	Library struct {
		Tracks      int64 `json:"tracks"`
		Albums      int64 `json:"albums"`
		Artists     int64 `json:"artists"`
		Playlists   int64 `json:"playlists"`
		Shares      int64 `json:"shares"`
		Radios      int64 `json:"radios"`
		ActiveUsers int64 `json:"activeUsers"`
	} `json:"library"`
	Config struct {
		LogLevel                   string        `json:"logLevel,omitempty"`
		LogFileConfigured          bool          `json:"logFileConfigured,omitempty"`
		TLSConfigured              bool          `json:"tlsConfigured,omitempty"`
		ScanSchedule               string        `json:"scanSchedule,omitempty"`
		TranscodingCacheSize       string        `json:"transcodingCacheSize,omitempty"`
		ImageCacheSize             string        `json:"imageCacheSize,omitempty"`
		EnableArtworkPrecache      bool          `json:"enableArtworkPrecache,omitempty"`
		EnableDownloads            bool          `json:"enableDownloads,omitempty"`
		EnableExternalServices     bool          `json:"enableExternalServices,omitempty"`
		EnableSharing              bool          `json:"enableSharing,omitempty"`
		EnableStarRating           bool          `json:"enableStarRating,omitempty"`
		EnableLastFM               bool          `json:"enableLastFM,omitempty"`
		EnableListenBrainz         bool          `json:"enableListenBrainz,omitempty"`
		EnableMediaFileCoverArt    bool          `json:"enableMediaFileCoverArt,omitempty"`
		EnableSpotify              bool          `json:"enableSpotify,omitempty"`
		EnableJukebox              bool          `json:"enableJukebox,omitempty"`
		EnablePrometheus           bool          `json:"enablePrometheus,omitempty"`
		SessionTimeout             time.Duration `json:"sessionTimeout,omitempty"`
		SearchFullString           bool          `json:"searchFullString,omitempty"`
		RecentlyAddedByModTime     bool          `json:"recentlyAddedByModTime,omitempty"`
		PreferSortTags             bool          `json:"preferSortTags,omitempty"`
		BackupSchedule             string        `json:"backupSchedule,omitempty"`
		BackupCount                int           `json:"backupCount,omitempty"`
		DefaultBackgroundURL       bool          `json:"defaultBackgroundURL,omitempty"`
		DevActivityPanel           bool          `json:"devActivityPanel,omitempty"`
		DevAutoLoginUsername       bool          `json:"devAutoLoginUsername,omitempty"`
		DevAutoCreateAdminPassword bool          `json:"devAutoCreateAdminPassword,omitempty"`
		EnableCoverAnimation       bool          `json:"enableCoverAnimation,omitempty"`
	} `json:"config"`
}

type FSInfo struct {
	Type string `json:"type,omitempty"`
}
