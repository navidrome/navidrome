package app

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
	policy := bluemonday.UGCPolicy()
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := ds.User(r.Context()).CountAll()
		firstTime := c == 0 && err == nil

		t, err := getIndexTemplate(r, fs)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		appConfig := map[string]interface{}{
			"version":                 consts.Version(),
			"firstTime":               firstTime,
			"baseURL":                 policy.Sanitize(strings.TrimSuffix(conf.Server.BaseURL, "/")),
			"loginBackgroundURL":      policy.Sanitize(conf.Server.UILoginBackgroundURL),
			"welcomeMessage":          policy.Sanitize(conf.Server.UIWelcomeMessage),
			"enableTranscodingConfig": conf.Server.EnableTranscodingConfig,
			"gaTrackingId":            conf.Server.GATrackingID,
			"enableDownloads":         conf.Server.EnableDownloads,
			"devActivityPanel":        conf.Server.DevActivityPanel,
			"devFastAccessCoverArt":   conf.Server.DevFastAccessCoverArt,
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
