package insights

type Data struct {
	InsightsID string `json:"id"`
	Version    string `json:"version"`
	Uptime     int64  `json:"uptime"`
	Build      struct {
		// build settings used by the Go compiler
		Settings  map[string]string `json:"settings"`
		GoVersion string            `json:"goVersion"`
	} `json:"build"`
	OS struct {
		Type          string `json:"type"`
		Distro        string `json:"distro,omitempty"`
		Version       string `json:"version,omitempty"`
		Containerized bool   `json:"containerized"`
		Arch          string `json:"arch"`
		NumCPU        int    `json:"numCPU"`
	} `json:"os"`
	Mem struct {
		Alloc      uint64 `json:"alloc"`
		TotalAlloc uint64 `json:"totalAlloc"`
		Sys        uint64 `json:"sys"`
		NumGC      uint32 `json:"numGC"`
	} `json:"mem"`
	FS struct {
		Music  *FSInfo `json:"music,omitempty"`
		Data   *FSInfo `json:"data,omitempty"`
		Cache  *FSInfo `json:"cache,omitempty"`
		Backup *FSInfo `json:"backup,omitempty"`
	} `json:"fs"`
	Library struct {
		Tracks        int64            `json:"tracks"`
		Albums        int64            `json:"albums"`
		Artists       int64            `json:"artists"`
		Playlists     int64            `json:"playlists"`
		Shares        int64            `json:"shares"`
		Radios        int64            `json:"radios"`
		Libraries     int64            `json:"libraries"`
		ActiveUsers   int64            `json:"activeUsers"`
		ActivePlayers map[string]int64 `json:"activePlayers,omitempty"`
	} `json:"library"`
	Config struct {
		LogLevel                string `json:"logLevel,omitempty"`
		LogFileConfigured       bool   `json:"logFileConfigured,omitempty"`
		TLSConfigured           bool   `json:"tlsConfigured,omitempty"`
		ScannerEnabled          bool   `json:"scannerEnabled,omitempty"`
		ScanSchedule            string `json:"scanSchedule,omitempty"`
		ScanWatcherWait         uint64 `json:"scanWatcherWait,omitempty"`
		ScanOnStartup           bool   `json:"scanOnStartup,omitempty"`
		TranscodingCacheSize    string `json:"transcodingCacheSize,omitempty"`
		ImageCacheSize          string `json:"imageCacheSize,omitempty"`
		EnableArtworkPrecache   bool   `json:"enableArtworkPrecache,omitempty"`
		EnableDownloads         bool   `json:"enableDownloads,omitempty"`
		EnableSharing           bool   `json:"enableSharing,omitempty"`
		EnableStarRating        bool   `json:"enableStarRating,omitempty"`
		EnableLastFM            bool   `json:"enableLastFM,omitempty"`
		EnableListenBrainz      bool   `json:"enableListenBrainz,omitempty"`
		EnableDeezer            bool   `json:"enableDeezer,omitempty"`
		EnableMediaFileCoverArt bool   `json:"enableMediaFileCoverArt,omitempty"`
		EnableSpotify           bool   `json:"enableSpotify,omitempty"`
		EnableJukebox           bool   `json:"enableJukebox,omitempty"`
		EnablePrometheus        bool   `json:"enablePrometheus,omitempty"`
		EnableCoverAnimation    bool   `json:"enableCoverAnimation,omitempty"`
		EnableNowPlaying        bool   `json:"enableNowPlaying,omitempty"`
		SessionTimeout          uint64 `json:"sessionTimeout,omitempty"`
		SearchFullString        bool   `json:"searchFullString,omitempty"`
		RecentlyAddedByModTime  bool   `json:"recentlyAddedByModTime,omitempty"`
		PreferSortTags          bool   `json:"preferSortTags,omitempty"`
		BackupSchedule          string `json:"backupSchedule,omitempty"`
		BackupCount             int    `json:"backupCount,omitempty"`
		DevActivityPanel        bool   `json:"devActivityPanel,omitempty"`
		DefaultBackgroundURLSet bool   `json:"defaultBackgroundURL,omitempty"`
		HasSmartPlaylists       bool   `json:"hasSmartPlaylists,omitempty"`
		ReverseProxyConfigured  bool   `json:"reverseProxyConfigured,omitempty"`
		HasCustomPID            bool   `json:"hasCustomPID,omitempty"`
		HasCustomTags           bool   `json:"hasCustomTags,omitempty"`
	} `json:"config"`
	Plugins map[string]PluginInfo `json:"plugins,omitempty"`
}

type PluginInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type FSInfo struct {
	Type string `json:"type,omitempty"`
}
