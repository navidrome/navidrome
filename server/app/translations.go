package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/resources"
	"github.com/navidrome/navidrome/utils"
)

type translation struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Data string `json:"data"`
}

var (
	once         sync.Once
	translations map[string]translation
)

func newTranslationRepository(context.Context) rest.Repository {
	dir := utils.NewMergeFS(
		http.FS(resources.Assets()),
		http.Dir(filepath.Join(conf.Server.DataFolder, "resources")),
	)
	if err := loadTranslations(dir); err != nil {
		log.Error("Error loading translation files", err)
	}
	return &translationRepository{}
}

type translationRepository struct{}

func (r *translationRepository) Read(id string) (interface{}, error) {
	if t, ok := translations[id]; ok {
		return t, nil
	}
	return nil, rest.ErrNotFound
}

// Simple Count implementation. Does not support any `options`
func (r *translationRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return int64(len(translations)), nil
}

// Simple ReadAll implementation, only returns IDs. Does not support any `options`
func (r *translationRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	var result []translation
	for _, t := range translations {
		t.Data = ""
		result = append(result, t)
	}
	return result, nil
}

func (r *translationRepository) EntityName() string {
	return "translation"
}

func (r *translationRepository) NewInstance() interface{} {
	return &translation{}
}

func loadTranslations(fs http.FileSystem) (loadError error) {
	once.Do(func() {
		translations = make(map[string]translation)
		dir, err := fs.Open(consts.I18nFolder)
		if err != nil {
			loadError = err
			return
		}
		files, err := dir.Readdir(0)
		if err != nil {
			loadError = err
			return
		}
		var languages []string
		for _, f := range files {
			t, err := loadTranslation(fs, f.Name())
			if err != nil {
				log.Error("Error loading translation file", "file", f.Name(), err)
				continue
			}
			translations[t.ID] = t
			languages = append(languages, t.ID)
		}
		log.Info("Loading translations", "languages", languages)
	})
	return
}

func loadTranslation(fs http.FileSystem, fileName string) (translation translation, err error) {
	// Get id and full path
	name := filepath.Base(fileName)
	id := strings.TrimSuffix(name, filepath.Ext(name))
	filePath := filepath.Join(consts.I18nFolder, name)

	// Load translation from json file
	file, err := fs.Open(filePath)
	if err != nil {
		return
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return
	}
	var out map[string]interface{}
	if err = json.Unmarshal(data, &out); err != nil {
		return
	}

	// Compress JSON
	buf := new(bytes.Buffer)
	if err = json.Compact(buf, data); err != nil {
		return
	}

	translation.Data = buf.String()
	translation.Name = out["languageName"].(string)
	translation.ID = id
	return
}

var _ rest.Repository = (*translationRepository)(nil)
