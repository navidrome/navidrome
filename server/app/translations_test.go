package app

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"path/filepath"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/resources"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Translations", func() {
	Describe("I18n files", func() {
		var fs http.FileSystem
		BeforeEach(func() {
			fs = http.FS(resources.Assets())
		})
		It("contains only valid json language files", func() {
			dir, _ := fs.Open(consts.I18nFolder)
			files, _ := dir.Readdir(0)
			for _, f := range files {
				name := filepath.Base(f.Name())
				filePath := filepath.Join(consts.I18nFolder, name)
				file, _ := fs.Open(filePath)
				data, _ := ioutil.ReadAll(file)
				var out map[string]interface{}

				Expect(filepath.Ext(filePath)).To(Equal(".json"), filePath)
				Expect(json.Unmarshal(data, &out)).To(BeNil(), filePath)
				Expect(out["languageName"]).ToNot(BeEmpty(), filePath)
			}
		})
	})

	Describe("loadTranslation", func() {
		It("loads a translation file correctly", func() {
			fs := http.Dir("ui/src")
			tr, err := loadTranslation(fs, "en.json")
			Expect(err).To(BeNil())
			Expect(tr.ID).To(Equal("en"))
			Expect(tr.Name).To(Equal("English"))
			var out map[string]interface{}
			Expect(json.Unmarshal([]byte(tr.Data), &out)).To(BeNil())
		})
	})
})
