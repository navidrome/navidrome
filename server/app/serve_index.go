package app

import (
	"encoding/json"
	"html/template"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
)

// Injects the config in the `index.html` template
func ServeIndex(ds model.DataStore, fs http.FileSystem) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := ds.User(r.Context()).CountAll()
		firstTime := c == 0 && err == nil

		t := getIndexTemplate(r, fs)

		if err != nil {
			log.Error("Error loading default English translation file", err)
		}
		appConfig := map[string]interface{}{
			"version":                 consts.Version(),
			"firstTime":               firstTime,
			"baseURL":                 strings.TrimSuffix(conf.Server.BaseURL, "/"),
			"loginBackgroundURL":      conf.Server.UILoginBackgroundURL,
			"enableTranscodingConfig": conf.Server.EnableTranscodingConfig,
		}
		j, err := json.Marshal(appConfig)
		if err != nil {
			log.Error(r, "Error converting config to JSON", "config", appConfig, err)
		} else {
			log.Trace(r, "Injecting config in index.html", "config", string(j))
		}

		data := map[string]interface{}{
			"AppConfig": string(j),
			"Version":   consts.Version(),
		}
		err = t.Execute(w, data)
		if err != nil {
			log.Error(r, "Could not execute `index.html` template", err)
		}
	}
}

func getIndexTemplate(r *http.Request, fs http.FileSystem) *template.Template {
	t := template.New("initial state")
	indexHtml, err := fs.Open("index.html")
	if err != nil {
		log.Error(r, "Could not find `index.html` template", err)
	}
	indexStr, err := ioutil.ReadAll(indexHtml)
	if err != nil {
		log.Error(r, "Could not read from `index.html`", err)
	}
	t, err = t.Parse(string(indexStr))
	if err != nil {
		log.Error(r, "Error parsing `index.html`", err)
	}
	return t
}
