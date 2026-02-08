package nativeapi

import (
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/resources"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Translations", func() {
	Describe("I18n files", func() {
		It("contains only valid json language files", func() {
			fsys := resources.FS()
			dir, _ := fsys.Open(consts.I18nFolder)
			files, _ := dir.(fs.ReadDirFile).ReadDir(-1)
			for _, f := range files {
				name := filepath.Base(f.Name())
				filePath := filepath.Join(consts.I18nFolder, name)
				file, _ := fsys.Open(filePath)
				data, _ := io.ReadAll(file)
				var out map[string]any

				Expect(filepath.Ext(filePath)).To(Equal(".json"), filePath)
				Expect(json.Unmarshal(data, &out)).To(BeNil(), filePath)
				Expect(out["languageName"]).ToNot(BeEmpty(), filePath)
			}
		})
	})

	Describe("loadTranslation", func() {
		It("loads a translation file correctly", func() {
			fs := os.DirFS("ui/src")
			tr, err := loadTranslation(fs, "en.json")
			Expect(err).To(BeNil())
			Expect(tr.ID).To(Equal("en"))
			Expect(tr.Name).To(Equal("English"))
			var out map[string]any
			Expect(json.Unmarshal([]byte(tr.Data), &out)).To(BeNil())
		})
	})
})
