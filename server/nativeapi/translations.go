package nativeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"path"
	"strings"
	"sync"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/resources"
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

func newTranslationRepository(ctx context.Context) rest.Repository {
	if err := loadTranslations(ctx, resources.FS()); err != nil {
		log.Error(ctx, "Error loading translation files", err)
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

// Count simple implementation, does not support any `options`
func (r *translationRepository) Count(...rest.QueryOptions) (int64, error) {
	return int64(len(translations)), nil
}

// ReadAll simple implementation, only returns IDs. Does not support any `options`
func (r *translationRepository) ReadAll(...rest.QueryOptions) (interface{}, error) {
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

func loadTranslations(ctx context.Context, fsys fs.FS) (loadError error) {
	once.Do(func() {
		translations = make(map[string]translation)
		dir, err := fsys.Open(consts.I18nFolder)
		if err != nil {
			loadError = err
			return
		}
		files, err := dir.(fs.ReadDirFile).ReadDir(-1)
		if err != nil {
			loadError = err
			return
		}
		var languages []string
		for _, f := range files {
			t, err := loadTranslation(fsys, f.Name())
			if err != nil {
				log.Error(ctx, "Error loading translation file", "file", f.Name(), err)
				continue
			}
			translations[t.ID] = t
			languages = append(languages, t.ID)
		}
		log.Info(ctx, "Loading translations", "languages", languages)
	})
	return
}

func loadTranslation(fsys fs.FS, fileName string) (translation translation, err error) {
	// Get id and full path
	name := path.Base(fileName)
	id := strings.TrimSuffix(name, path.Ext(name))
	filePath := path.Join(consts.I18nFolder, name)

	// Load translation from json file
	file, err := fsys.Open(filePath)
	if err != nil {
		return
	}
	data, err := io.ReadAll(file)
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
