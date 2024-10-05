package metrics

import (
	"context"
	"encoding/json"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type data struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Uptime  int64  `json:"uptime"`
	Build   struct {
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
		Tracks    int `json:"tracks"`
		Albums    int `json:"albums"`
		Artists   int `json:"artists"`
		Playlists int `json:"playlists"`
		Shares    int `json:"shares"`
		Radios    int `json:"radios"`
	} `json:"library"`
}

type Insights interface {
	Collect(ctx context.Context) string
}

var insightsID string

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
		ID:      insightsID,
		Version: consts.Version,
	}

	// Build info
	data.Build.Settings, data.Build.GoVersion = buildInfo()

	// OS info
	data.OS.Type = runtime.GOOS
	data.OS.Arch = runtime.GOARCH
	data.OS.NumCPU = runtime.NumCPU()
	data.OS.Version, data.OS.Distro = getOSVersion()

	return data
})

func (s insights) Collect(ctx context.Context) string {
	data := staticData()
	data.Uptime = time.Since(consts.ServerStart).Milliseconds() / 1000

	// TODO Library info

	// Marshal to JSON
	resp, err := json.Marshal(data)
	if err != nil {
		log.Trace(ctx, "Could not marshal Insights data", err)
		return ""
	}
	return string(resp)
}
