package app

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/resources"
	"github.com/deluan/rest"
)

const i18nFolder = "i18n"

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
	if err := loadTranslations(); err != nil {
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

// Simple ReadAll implementation, only returns IDs. Does not support any `options`, always sort by name asc
func (r *translationRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	var result []translation
	for _, t := range translations {
		t.Data = ""
		result = append(result, t)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, nil
}

func (r *translationRepository) EntityName() string {
	return "translation"
}

func (r *translationRepository) NewInstance() interface{} {
	return &translation{}
}

func loadTranslations() (loadError error) {
	once.Do(func() {
		translations = make(map[string]translation)
		dir, err := resources.AssetFile().Open(i18nFolder)
		if err != nil {
			loadError = err
			return
		}
		files, err := dir.Readdir(0)
		if err != nil {
			loadError = err
			return
		}
		for _, f := range files {
			t, err := loadTranslation(f.Name())
			if err != nil {
				log.Error("Error loading translation file", "file", f.Name(), err)
				continue
			}
			translations[t.ID] = t
		}
	})
	return
}

func loadTranslation(fileName string) (trans translation, err error) {
	id := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	filePath := filepath.Join(i18nFolder, fileName)
	data, err := resources.Asset(filePath)
	trans.Data = string(data)
	if err != nil {
		return
	}
	var out map[string]interface{}
	err = json.Unmarshal(data, &out)
	if err != nil {
		return
	}
	trans.Name = out["languageName"].(string)
	trans.ID = id
	return
}

var _ rest.Repository = (*translationRepository)(nil)
