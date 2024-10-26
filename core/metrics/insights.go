package metrics

import (
	"context"
	"encoding/json"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"golang.org/x/time/rate"
)

type data struct {
	InsightsID string `json:"id"`
	Version    string `json:"version"`
	Uptime     int64  `json:"uptime"`
	Build      struct {
		Settings  map[string]string `json:"settings"`
		GoVersion string            `json:"goVersion"`
	} `json:"build"`
	OS struct {
		Type    string `json:"type"`
		Distro  string `json:"distro"`
		Version string `json:"version"`
		Arch    string `json:"arch"`
		NumCPU  int    `json:"numCPU"`
	} `json:"os"`
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
		LogLevel                string        `json:"logLevel,omitempty"`
		LogFileConfigured       bool          `json:"logFileConfigured,omitempty"`
		TLSConfigured           bool          `json:"tlsConfigured,omitempty"`
		ScanSchedule            string        `json:"scanSchedule,omitempty"`
		TranscodingCacheSize    string        `json:"transcodingCacheSize,omitempty"`
		ImageCacheSize          string        `json:"imageCacheSize,omitempty"`
		EnableArtworkPrecache   bool          `json:"enableArtworkPrecache,omitempty"`
		EnableDownloads         bool          `json:"enableDownloads,omitempty"`
		EnableExternalServices  bool          `json:"enableExternalServices,omitempty"`
		EnableSharing           bool          `json:"enableSharing,omitempty"`
		EnableStarRating        bool          `json:"enableStarRating,omitempty"`
		EnableLastFM            bool          `json:"enableLastFM,omitempty"`
		EnableListenBrainz      bool          `json:"enableListenBrainz,omitempty"`
		EnableMediaFileCoverArt bool          `json:"enableMediaFileCoverArt,omitempty"`
		EnableSpotify           bool          `json:"enableSpotify,omitempty"`
		EnableJukebox           bool          `json:"enableJukebox,omitempty"`
		EnablePrometheus        bool          `json:"enablePrometheus,omitempty"`
		SessionTimeout          time.Duration `json:"sessionTimeout,omitempty"`
		SearchFullString        bool          `json:"searchFullString,omitempty"`
		RecentlyAddedByModTime  bool          `json:"recentlyAddedByModTime,omitempty"`
		PreferSortTags          bool          `json:"preferSortTags,omitempty"`
		BackupSchedule          string        `json:"backupSchedule,omitempty"`
		BackupCount             int           `json:"backupCount,omitempty"`
	} `json:"config"`
}

type Insights interface {
	Collect(ctx context.Context) string
}

var (
	insightsID    string
	libraryUpdate = rate.Sometimes{Interval: 10 * time.Minute}
)

type insights struct {
	ds model.DataStore
}

func NewInsights(ds model.DataStore) Insights {
	id, err := ds.Property(context.TODO()).Get(consts.InsightsID)
	if err != nil {
		log.Trace("Could not get Insights ID from DB", err)
		id = uuid.NewString()
		err = ds.Property(context.TODO()).Put(consts.InsightsID, id)
		if err != nil {
			log.Trace("Could not save Insights ID to DB", err)
		}
	}
	insightsID = id
	return &insights{ds: ds}
}

func buildInfo() (map[string]string, string) {
	bInfo := map[string]string{}
	var version string
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Value == "" {
				continue
			}
			bInfo[setting.Key] = setting.Value
		}
		version = info.GoVersion
	}
	return bInfo, version
}

var staticData = sync.OnceValue(func() data {
	// Basic info
	data := data{
		InsightsID: insightsID,
		Version:    consts.Version,
	}

	// Build info
	data.Build.Settings, data.Build.GoVersion = buildInfo()

	// OS info
	data.OS.Type = runtime.GOOS
	data.OS.Arch = runtime.GOARCH
	data.OS.NumCPU = runtime.NumCPU()
	data.OS.Version, data.OS.Distro = getOSVersion()

	// Config info
	data.Config.LogLevel = conf.Server.LogLevel
	data.Config.LogFileConfigured = conf.Server.LogFile != ""
	data.Config.TLSConfigured = conf.Server.TLSCert != "" && conf.Server.TLSKey != ""
	data.Config.EnableArtworkPrecache = conf.Server.EnableArtworkPrecache
	data.Config.EnableDownloads = conf.Server.EnableDownloads
	data.Config.EnableExternalServices = conf.Server.EnableExternalServices
	data.Config.EnableSharing = conf.Server.EnableSharing
	data.Config.EnableStarRating = conf.Server.EnableStarRating
	data.Config.EnableLastFM = conf.Server.LastFM.Enabled
	data.Config.EnableListenBrainz = conf.Server.ListenBrainz.Enabled
	data.Config.EnableMediaFileCoverArt = conf.Server.EnableMediaFileCoverArt
	data.Config.EnableSpotify = conf.Server.Spotify.ID != ""
	data.Config.EnableJukebox = conf.Server.Jukebox.Enabled
	data.Config.EnablePrometheus = conf.Server.Prometheus.Enabled
	data.Config.TranscodingCacheSize = conf.Server.TranscodingCacheSize
	data.Config.ImageCacheSize = conf.Server.ImageCacheSize
	data.Config.ScanSchedule = conf.Server.ScanSchedule
	data.Config.SessionTimeout = conf.Server.SessionTimeout
	data.Config.SearchFullString = conf.Server.SearchFullString
	data.Config.RecentlyAddedByModTime = conf.Server.RecentlyAddedByModTime
	data.Config.PreferSortTags = conf.Server.PreferSortTags
	data.Config.BackupSchedule = conf.Server.Backup.Schedule
	data.Config.BackupCount = conf.Server.Backup.Count

	return data
})

func (s insights) Collect(ctx context.Context) string {
	data := staticData()
	data.Uptime = time.Since(consts.ServerStart).Milliseconds() / 1000
	libraryUpdate.Do(func() {
		data.Library.Tracks, _ = s.ds.MediaFile(ctx).CountAll()
		data.Library.Albums, _ = s.ds.Album(ctx).CountAll()
		data.Library.Artists, _ = s.ds.Artist(ctx).CountAll()
		data.Library.Playlists, _ = s.ds.Playlist(ctx).Count()
		data.Library.Shares, _ = s.ds.Share(ctx).CountAll()
		data.Library.Radios, _ = s.ds.Radio(ctx).Count()
		data.Library.ActiveUsers, _ = s.ds.User(ctx).CountAll(model.QueryOptions{
			Filters: squirrel.Gt{"last_access_at": time.Now().Add(-7 * 24 * time.Hour)},
		})
	})

	// Marshal to JSON
	resp, err := json.Marshal(data)
	if err != nil {
		log.Trace(ctx, "Could not marshal Insights data", err)
		return ""
	}
	return string(resp)
}
