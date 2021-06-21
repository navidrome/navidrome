package server

import (
	"encoding/json"
	"html/template"
	"io/fs"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// Injects the config in the `index.html` template
func serveIndex(ds model.DataStore, fs fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := ds.User(r.Context()).CountAll()
		firstTime := c == 0 && err == nil

		t, err := getIndexTemplate(r, fs)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		policy := bluemonday.UGCPolicy()
		appConfig := map[string]interface{}{
			"version":                 consts.Version(),
			"firstTime":               firstTime,
			"baseURL":                 policy.Sanitize(strings.TrimSuffix(conf.Server.BaseURL, "/")),
			"loginBackgroundURL":      policy.Sanitize(conf.Server.UILoginBackgroundURL),
			"welcomeMessage":          policy.Sanitize(conf.Server.UIWelcomeMessage),
			"enableTranscodingConfig": conf.Server.EnableTranscodingConfig,
			"enableDownloads":         conf.Server.EnableDownloads,
			"enableFavourites":        conf.Server.EnableFavourites,
			"enableStarRating":        conf.Server.EnableStarRating,
			"defaultTheme":            conf.Server.DefaultTheme,
			"gaTrackingId":            conf.Server.GATrackingID,
			"losslessFormats":         strings.ToUpper(strings.Join(consts.LosslessFormats, ",")),
			"devActivityPanel":        conf.Server.DevActivityPanel,
			"devFastAccessCoverArt":   conf.Server.DevFastAccessCoverArt,
			"enableUserEditing":       conf.Server.EnableUserEditing,
			"devEnableShare":          conf.Server.DevEnableShare,
			"devEnableScrobble":       conf.Server.DevEnableScrobble,
			"lastFMApiKey":            conf.Server.LastFM.ApiKey,
		}
		auth := handleLoginFromHeaders(ds, r)
		if auth != nil {
			appConfig["auth"] = auth
		}
		j, err := json.Marshal(appConfig)
		if err != nil {
			log.Error(r, "Error converting config to JSON", "config", appConfig, err)
		} else {
			log.Trace(r, "Injecting config in index.html", "config", string(j))
		}

		log.Debug("UI configuration", "appConfig", appConfig)
		version := consts.Version()
		if version != "dev" {
			version = "v" + version
		}
		data := map[string]interface{}{
			"AppConfig": string(j),
			"Version":   version,
		}
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
	indexStr, err := ioutil.ReadAll(indexHtml)
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
