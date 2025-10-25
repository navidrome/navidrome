package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"net/http"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/metrics/insights"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/plugins/schema"
	"github.com/navidrome/navidrome/utils/singleton"
)

type Insights interface {
	Run(ctx context.Context)
	LastRun(ctx context.Context) (timestamp time.Time, success bool)
}

var (
	insightsID string
)

type insightsCollector struct {
	ds           model.DataStore
	pluginLoader PluginLoader
	lastRun      atomic.Int64
	lastStatus   atomic.Bool
}

// PluginLoader defines an interface for loading plugins
type PluginLoader interface {
	PluginList() map[string]schema.PluginManifest
}

func GetInstance(ds model.DataStore, pluginLoader PluginLoader) Insights {
	return singleton.GetInstance(func() *insightsCollector {
		id, err := ds.Property(context.TODO()).Get(consts.InsightsIDKey)
		if err != nil {
			log.Trace("Could not get Insights ID from DB. Creating one", err)
			id = uuid.NewString()
			err = ds.Property(context.TODO()).Put(consts.InsightsIDKey, id)
			if err != nil {
				log.Trace("Could not save Insights ID to DB", err)
			}
		}
		insightsID = id
		return &insightsCollector{ds: ds, pluginLoader: pluginLoader}
	})
}

func (c *insightsCollector) Run(ctx context.Context) {
	ctx = auth.WithAdminUser(ctx, c.ds)
	for {
		c.sendInsights(ctx)
		select {
		case <-time.After(consts.InsightsUpdateInterval):
			continue
		case <-ctx.Done():
			return
		}
	}
}

func (c *insightsCollector) LastRun(context.Context) (timestamp time.Time, success bool) {
	t := c.lastRun.Load()
	return time.UnixMilli(t), c.lastStatus.Load()
}

func (c *insightsCollector) sendInsights(ctx context.Context) {
	count, err := c.ds.User(ctx).CountAll(model.QueryOptions{})
	if err != nil {
		log.Trace(ctx, "Could not check user count", err)
		return
	}
	if count == 0 {
		log.Trace(ctx, "No users found, skipping Insights data collection")
		return
	}
	hc := &http.Client{
		Timeout: consts.DefaultHttpClientTimeOut,
	}
	data := c.collect(ctx)
	if data == nil {
		return
	}
	body := bytes.NewReader(data)
	req, err := http.NewRequestWithContext(ctx, "POST", consts.InsightsEndpoint, body)
	if err != nil {
		log.Trace(ctx, "Could not create Insights request", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := hc.Do(req)
	if err != nil {
		log.Trace(ctx, "Could not send Insights data", err)
		return
	}
	log.Info(ctx, "Sent Insights data (for details see http://navidrome.org/docs/getting-started/insights", "data",
		string(data), "server", consts.InsightsEndpoint, "status", resp.Status)
	c.lastRun.Store(time.Now().UnixMilli())
	c.lastStatus.Store(resp.StatusCode < 300)
	resp.Body.Close()
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

func getFSInfo(path string) *insights.FSInfo {
	var info insights.FSInfo

	// Normalize the path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil
	}
	absPath = filepath.Clean(absPath)

	fsType, err := getFilesystemType(absPath)
	if err != nil {
		return nil
	}
	info.Type = fsType
	return &info
}

var staticData = sync.OnceValue(func() insights.Data {
	// Basic info
	data := insights.Data{
		InsightsID: insightsID,
		Version:    consts.Version,
	}

	// Build info
	data.Build.Settings, data.Build.GoVersion = buildInfo()
	data.OS.Containerized = consts.InContainer

	// OS info
	data.OS.Type = runtime.GOOS
	data.OS.Arch = runtime.GOARCH
	data.OS.NumCPU = runtime.NumCPU()
	data.OS.Version, data.OS.Distro = getOSVersion()

	// FS info
	data.FS.Music = getFSInfo(conf.Server.MusicFolder)
	data.FS.Data = getFSInfo(conf.Server.DataFolder)
	if conf.Server.CacheFolder != "" {
		data.FS.Cache = getFSInfo(conf.Server.CacheFolder)
	}
	if conf.Server.Backup.Path != "" {
		data.FS.Backup = getFSInfo(conf.Server.Backup.Path)
	}

	// Config info
	data.Config.LogLevel = conf.Server.LogLevel
	data.Config.LogFileConfigured = conf.Server.LogFile != ""
	data.Config.TLSConfigured = conf.Server.TLSCert != "" && conf.Server.TLSKey != ""
	data.Config.DefaultBackgroundURLSet = conf.Server.UILoginBackgroundURL == consts.DefaultUILoginBackgroundURL
	data.Config.EnableArtworkPrecache = conf.Server.EnableArtworkPrecache
	data.Config.EnableCoverAnimation = conf.Server.EnableCoverAnimation
	data.Config.EnableNowPlaying = conf.Server.EnableNowPlaying
	data.Config.EnableDownloads = conf.Server.EnableDownloads
	data.Config.EnableSharing = conf.Server.EnableSharing
	data.Config.EnableStarRating = conf.Server.EnableStarRating
	data.Config.EnableLastFM = conf.Server.LastFM.Enabled && conf.Server.LastFM.ApiKey != "" && conf.Server.LastFM.Secret != ""
	data.Config.EnableSpotify = conf.Server.Spotify.ID != "" && conf.Server.Spotify.Secret != ""
	data.Config.EnableListenBrainz = conf.Server.ListenBrainz.Enabled
	data.Config.EnableDeezer = conf.Server.Deezer.Enabled
	data.Config.EnableMediaFileCoverArt = conf.Server.EnableMediaFileCoverArt
	data.Config.EnableJukebox = conf.Server.Jukebox.Enabled
	data.Config.EnablePrometheus = conf.Server.Prometheus.Enabled
	data.Config.TranscodingCacheSize = conf.Server.TranscodingCacheSize
	data.Config.ImageCacheSize = conf.Server.ImageCacheSize
	data.Config.SessionTimeout = uint64(math.Trunc(conf.Server.SessionTimeout.Seconds()))
	data.Config.SearchFullString = conf.Server.SearchFullString
	data.Config.RecentlyAddedByModTime = conf.Server.RecentlyAddedByModTime
	data.Config.PreferSortTags = conf.Server.PreferSortTags
	data.Config.BackupSchedule = conf.Server.Backup.Schedule
	data.Config.BackupCount = conf.Server.Backup.Count
	data.Config.DevActivityPanel = conf.Server.DevActivityPanel
	data.Config.ScannerEnabled = conf.Server.Scanner.Enabled
	data.Config.ScanSchedule = conf.Server.Scanner.Schedule
	data.Config.ScanWatcherWait = uint64(math.Trunc(conf.Server.Scanner.WatcherWait.Seconds()))
	data.Config.ScanOnStartup = conf.Server.Scanner.ScanOnStartup
	data.Config.ReverseProxyConfigured = conf.Server.ReverseProxyWhitelist != ""
	data.Config.HasCustomPID = conf.Server.PID.Track != "" || conf.Server.PID.Album != ""
	data.Config.HasCustomTags = len(conf.Server.Tags) > 0

	return data
})

func (c *insightsCollector) collect(ctx context.Context) []byte {
	data := staticData()
	data.Uptime = time.Since(consts.ServerStart).Milliseconds() / 1000

	// Library info
	var err error
	data.Library.Tracks, err = c.ds.MediaFile(ctx).CountAll()
	if err != nil {
		log.Trace(ctx, "Error reading tracks count", err)
	}
	data.Library.Albums, err = c.ds.Album(ctx).CountAll()
	if err != nil {
		log.Trace(ctx, "Error reading albums count", err)
	}
	data.Library.Artists, err = c.ds.Artist(ctx).CountAll()
	if err != nil {
		log.Trace(ctx, "Error reading artists count", err)
	}
	data.Library.Playlists, err = c.ds.Playlist(ctx).CountAll()
	if err != nil {
		log.Trace(ctx, "Error reading playlists count", err)
	}
	data.Library.Shares, err = c.ds.Share(ctx).CountAll()
	if err != nil {
		log.Trace(ctx, "Error reading shares count", err)
	}
	data.Library.Radios, err = c.ds.Radio(ctx).Count()
	if err != nil {
		log.Trace(ctx, "Error reading radios count", err)
	}
	data.Library.Libraries, err = c.ds.Library(ctx).CountAll()
	if err != nil {
		log.Trace(ctx, "Error reading libraries count", err)
	}
	data.Library.ActiveUsers, err = c.ds.User(ctx).CountAll(model.QueryOptions{
		Filters: squirrel.Gt{"last_access_at": time.Now().Add(-7 * 24 * time.Hour)},
	})
	if err != nil {
		log.Trace(ctx, "Error reading active users count", err)
	}

	// Check for smart playlists
	data.Config.HasSmartPlaylists, err = c.hasSmartPlaylists(ctx)
	if err != nil {
		log.Trace(ctx, "Error checking for smart playlists", err)
	}

	// Collect plugins if permitted and enabled
	if conf.Server.DevEnablePluginsInsights && conf.Server.Plugins.Enabled {
		data.Plugins = c.collectPlugins(ctx)
	}

	// Collect active players if permitted
	if conf.Server.DevEnablePlayerInsights {
		data.Library.ActivePlayers, err = c.ds.Player(ctx).CountByClient(model.QueryOptions{
			Filters: squirrel.Gt{"last_seen": time.Now().Add(-7 * 24 * time.Hour)},
		})
		if err != nil {
			log.Trace(ctx, "Error reading active players count", err)
		}
	}

	// Memory info
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	data.Mem.Alloc = m.Alloc
	data.Mem.TotalAlloc = m.TotalAlloc
	data.Mem.Sys = m.Sys
	data.Mem.NumGC = m.NumGC

	// Marshal to JSON
	resp, err := json.Marshal(data)
	if err != nil {
		log.Trace(ctx, "Could not marshal Insights data", err)
		return nil
	}
	return resp
}

// hasSmartPlaylists checks if there are any smart playlists (playlists with rules)
func (c *insightsCollector) hasSmartPlaylists(ctx context.Context) (bool, error) {
	count, err := c.ds.Playlist(ctx).CountAll(model.QueryOptions{
		Filters: squirrel.And{squirrel.NotEq{"rules": ""}, squirrel.NotEq{"rules": nil}},
	})
	return count > 0, err
}

// collectPlugins collects information about installed plugins
func (c *insightsCollector) collectPlugins(_ context.Context) map[string]insights.PluginInfo {
	plugins := make(map[string]insights.PluginInfo)
	for id, manifest := range c.pluginLoader.PluginList() {
		plugins[id] = insights.PluginInfo{
			Name:    manifest.Name,
			Version: manifest.Version,
		}
	}
	return plugins
}
