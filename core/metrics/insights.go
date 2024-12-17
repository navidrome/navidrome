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
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/metrics/insights"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/singleton"
	"golang.org/x/time/rate"
)

type Insights interface {
	Run(ctx context.Context)
	LastRun(ctx context.Context) (timestamp time.Time, success bool)
}

var (
	insightsID    string
	libraryUpdate = rate.Sometimes{Interval: 10 * time.Minute}
)

type insightsCollector struct {
	ds         model.DataStore
	lastRun    time.Time
	lastStatus bool
}

func GetInstance(ds model.DataStore) Insights {
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
		return &insightsCollector{ds: ds}
	})
}

func (c *insightsCollector) Run(ctx context.Context) {
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
	return c.lastRun, c.lastStatus
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
	c.lastRun = time.Now()
	c.lastStatus = resp.StatusCode < 300
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
	data.Config.EnableDownloads = conf.Server.EnableDownloads
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
	data.Config.SessionTimeout = uint64(math.Trunc(conf.Server.SessionTimeout.Seconds()))
	data.Config.SearchFullString = conf.Server.SearchFullString
	data.Config.RecentlyAddedByModTime = conf.Server.RecentlyAddedByModTime
	data.Config.PreferSortTags = conf.Server.PreferSortTags
	data.Config.BackupSchedule = conf.Server.Backup.Schedule
	data.Config.BackupCount = conf.Server.Backup.Count
	data.Config.DevActivityPanel = conf.Server.DevActivityPanel

	return data
})

func (c *insightsCollector) collect(ctx context.Context) []byte {
	data := staticData()
	data.Uptime = time.Since(consts.ServerStart).Milliseconds() / 1000
	libraryUpdate.Do(func() {
		data.Library.Tracks, _ = c.ds.MediaFile(ctx).CountAll()
		data.Library.Albums, _ = c.ds.Album(ctx).CountAll()
		data.Library.Artists, _ = c.ds.Artist(ctx).CountAll()
		data.Library.Playlists, _ = c.ds.Playlist(ctx).Count()
		data.Library.Shares, _ = c.ds.Share(ctx).CountAll()
		data.Library.Radios, _ = c.ds.Radio(ctx).Count()
		data.Library.ActiveUsers, _ = c.ds.User(ctx).CountAll(model.QueryOptions{
			Filters: squirrel.Gt{"last_access_at": time.Now().Add(-7 * 24 * time.Hour)},
		})
	})
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