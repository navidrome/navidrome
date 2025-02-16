package server

import (
	"encoding/json"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/mime"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/navidrome/navidrome/utils/str"
)

func Index(ds model.DataStore, fs fs.FS) http.HandlerFunc {
	return serveIndex(ds, fs, nil)
}

func IndexWithShare(ds model.DataStore, fs fs.FS, shareInfo *model.Share) http.HandlerFunc {
	return serveIndex(ds, fs, shareInfo)
}

// Injects the config in the `index.html` template
func serveIndex(ds model.DataStore, fs fs.FS, shareInfo *model.Share) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := ds.User(r.Context()).CountAll()
		firstTime := c == 0 && err == nil

		t, err := getIndexTemplate(r, fs)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		appConfig := map[string]interface{}{
			"version":                   consts.Version,
			"firstTime":                 firstTime,
			"variousArtistsId":          consts.VariousArtistsID,
			"baseURL":                   str.SanitizeText(strings.TrimSuffix(conf.Server.BasePath, "/")),
			"loginBackgroundURL":        str.SanitizeText(conf.Server.UILoginBackgroundURL),
			"welcomeMessage":            str.SanitizeText(conf.Server.UIWelcomeMessage),
			"maxSidebarPlaylists":       conf.Server.MaxSidebarPlaylists,
			"enableTranscodingConfig":   conf.Server.EnableTranscodingConfig,
			"enableDownloads":           conf.Server.EnableDownloads,
			"enableFavourites":          conf.Server.EnableFavourites,
			"enableStarRating":          conf.Server.EnableStarRating,
			"defaultTheme":              conf.Server.DefaultTheme,
			"defaultLanguage":           conf.Server.DefaultLanguage,
			"defaultUIVolume":           conf.Server.DefaultUIVolume,
			"enableCoverAnimation":      conf.Server.EnableCoverAnimation,
			"gaTrackingId":              conf.Server.GATrackingID,
			"losslessFormats":           strings.ToUpper(strings.Join(mime.LosslessFormats, ",")),
			"devActivityPanel":          conf.Server.DevActivityPanel,
			"enableUserEditing":         conf.Server.EnableUserEditing,
			"enableSharing":             conf.Server.EnableSharing,
			"shareURL":                  conf.Server.ShareURL,
			"defaultDownloadableShare":  conf.Server.DefaultDownloadableShare,
			"devSidebarPlaylists":       conf.Server.DevSidebarPlaylists,
			"lastFMEnabled":             conf.Server.LastFM.Enabled,
			"devShowArtistPage":         conf.Server.DevShowArtistPage,
			"listenBrainzEnabled":       conf.Server.ListenBrainz.Enabled,
			"enableExternalServices":    conf.Server.EnableExternalServices,
			"enableReplayGain":          conf.Server.EnableReplayGain,
			"defaultDownsamplingFormat": conf.Server.DefaultDownsamplingFormat,
			"separator":                 string(os.PathSeparator),
			"enableInspect":             conf.Server.Inspect.Enabled,
		}
		if strings.HasPrefix(conf.Server.UILoginBackgroundURL, "/") {
			appConfig["loginBackgroundURL"] = path.Join(conf.Server.BasePath, conf.Server.UILoginBackgroundURL)
		}
		auth := handleLoginFromHeaders(ds, r)
		if auth != nil {
			appConfig["auth"] = auth
		}
		appConfigJson, err := json.Marshal(appConfig)
		if err != nil {
			log.Error(r, "Error converting config to JSON", "config", appConfig, err)
		} else {
			log.Trace(r, "Injecting config in index.html", "config", string(appConfigJson))
		}

		log.Debug("UI configuration", "appConfig", appConfig)
		version := consts.Version
		if version != "dev" {
			version = "v" + version
		}
		data := map[string]interface{}{
			"AppConfig": string(appConfigJson),
			"Version":   version,
		}
		addShareData(r, data, shareInfo)

		w.Header().Set("Content-Type", "text/html")
		err = t.Execute(w, data)
		if err != nil {
			log.Error(r, "Could not execute `index.html` template", err)
		}
	}
}

func getIndexTemplate(r *http.Request, fs fs.FS) (*template.Template, error) {
	t := template.New("initial state")
	indexHtml, err := fs.Open("index.html")
	if err != nil {
		log.Error(r, "Could not find `index.html` template", err)
		return nil, err
	}
	indexStr, err := io.ReadAll(indexHtml)
	if err != nil {
		log.Error(r, "Could not read from `index.html`", err)
		return nil, err
	}
	t, err = t.Parse(string(indexStr))
	if err != nil {
		log.Error(r, "Error parsing `index.html`", err)
		return nil, err
	}
	return t, nil
}

type shareData struct {
	ID           string       `json:"id"`
	Description  string       `json:"description"`
	Downloadable bool         `json:"downloadable"`
	Tracks       []shareTrack `json:"tracks"`
}

type shareTrack struct {
	ID        string    `json:"id,omitempty"`
	Title     string    `json:"title,omitempty"`
	Artist    string    `json:"artist,omitempty"`
	Album     string    `json:"album,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
	Duration  float32   `json:"duration,omitempty"`
}

func addShareData(r *http.Request, data map[string]interface{}, shareInfo *model.Share) {
	ctx := r.Context()
	if shareInfo == nil || shareInfo.ID == "" {
		return
	}
	sd := shareData{
		ID:           shareInfo.ID,
		Description:  shareInfo.Description,
		Downloadable: shareInfo.Downloadable,
	}
	sd.Tracks = slice.Map(shareInfo.Tracks, func(mf model.MediaFile) shareTrack {
		return shareTrack{
			ID:        mf.ID,
			Title:     mf.Title,
			Artist:    mf.Artist,
			Album:     mf.Album,
			Duration:  mf.Duration,
			UpdatedAt: mf.UpdatedAt,
		}
	})

	shareInfoJson, err := json.Marshal(sd)
	if err != nil {
		log.Error(ctx, "Error converting shareInfo to JSON", "config", shareInfo, err)
	} else {
		log.Trace(ctx, "Injecting shareInfo in index.html", "config", string(shareInfoJson))
	}

	if shareInfo.Description != "" {
		data["ShareDescription"] = shareInfo.Description
	} else {
		data["ShareDescription"] = shareInfo.Contents
	}
	data["ShareURL"] = shareInfo.URL
	data["ShareImageURL"] = shareInfo.ImageURL
	data["ShareInfo"] = string(shareInfoJson)
}
