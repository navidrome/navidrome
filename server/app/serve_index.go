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

// Injects the `firstTime` config in the `index.html` template
func ServeIndex(ds model.DataStore, fs http.FileSystem) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := ds.User(r.Context()).CountAll()
		firstTime := c == 0 && err == nil

		t := template.New("initial state")
		indexHtml, err := fs.Open("index.html")
		if err != nil {
			log.Error(r, "Could not find `index.html` template", err)
		}
		indexStr, err := ioutil.ReadAll(indexHtml)
		if err != nil {
			log.Error(r, "Could not read from `index.html`", err)
		}
		t, _ = t.Parse(string(indexStr))
		appConfig := map[string]interface{}{
			"firstTime": firstTime,
			"baseURL":   strings.TrimSuffix(conf.Server.BaseURL, "/"),
		}
		j, _ := json.Marshal(appConfig)
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
