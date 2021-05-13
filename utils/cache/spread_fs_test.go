package cache

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Spread FS", func() {
	var fs *spreadFS
	var rootDir string

	BeforeEach(func() {
		var err error
		rootDir, _ = ioutil.TempDir("", "spread_fs")
		fs, err = NewSpreadFS(rootDir, 0755)
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(rootDir)).To(BeNil())
	})

	Describe("KeyMapper", func() {
		It("creates a file with proper name format", func() {
			mapped := fs.KeyMapper("abc")
			Expect(mapped).To(HavePrefix(fs.root))
			mapped = strings.TrimPrefix(mapped, fs.root)
			parts := strings.Split(mapped, string(filepath.Separator))
			Expect(parts).To(HaveLen(4))
			Expect(parts[3]).To(HaveLen(40))
		})
	})

	Describe("Reload", func() {
		var files []string

		BeforeEach(func() {
			files = []string{"aaaaa", "bbbbb", "ccccc"}
			for _, content := range files {
				file := fs.KeyMapper(content)
				f, err := fs.Create(file)
				Expect(err).To(BeNil())
				_, _ = f.Write([]byte(content))
				_ = f.Close()
			}
		})

		It("loads all files from fs", func() {
			var actual []string
			err := fs.Reload(func(key string, name string) {
				Expect(key).To(Equal(name))
				data, err := ioutil.ReadFile(name)
				Expect(err).To(BeNil())
				actual = append(actual, string(data))
			})
			Expect(err).To(BeNil())
			Expect(actual).To(HaveLen(len(files)))
			Expect(actual).To(ContainElements(files[0], files[1], files[2]))
		})
	})
})
