package app

import (
	"encoding/json"
	"html/template"
	"io/ioutil"
	"net/http"

	"github.com/deluan/navidrome/assets"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
)

// Injects the `firstTime` config in the `index.html` template
func ServeIndex(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := ds.User(r.Context()).CountAll()
		firstTime := c == 0 && err == nil

		t := template.New("initial state")
		fs := assets.AssetFile()
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
		}
		j, _ := json.Marshal(appConfig)
		data := map[string]interface{}{
			"AppConfig": string(j),
		}
		err = t.Execute(w, data)
		if err != nil {
			log.Error(r, "Could not execute `index.html` template", err)
		}
	}
}
