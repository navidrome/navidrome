package utils_test

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/deluan/navidrome/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("mergeFS", func() {
	var baseName, overlayName string
	var baseDir, overlayDir, mergedDir http.FileSystem

	BeforeEach(func() {
		baseName, _ = ioutil.TempDir("", "merge_fs_base_test")
		overlayName, _ = ioutil.TempDir("", "merge_fs_overlay_test")
		baseDir = http.Dir(baseName)
		overlayDir = http.Dir(overlayName)
		mergedDir = utils.NewMergeFS(baseDir, overlayDir)
	})

	It("reads from base dir if not found in overlay", func() {
		_f(baseName, "a.json")
		file, err := mergedDir.Open("a.json")
		Expect(err).To(BeNil())

		stat, err := file.Stat()
		Expect(err).To(BeNil())

		Expect(stat.Name()).To(Equal("a.json"))
	})

	It("reads overridden file", func() {
		_f(baseName, "b.json", "original")
		_f(baseName, "b.json", "overridden")

		file, err := mergedDir.Open("b.json")
		Expect(err).To(BeNil())

		content, err := ioutil.ReadAll(file)
		Expect(err).To(BeNil())
		Expect(string(content)).To(Equal("overridden"))
	})

	It("reads only files from base if overlay is empty", func() {
		_f(baseName, "test.txt")

		dir, err := mergedDir.Open(".")
		Expect(err).To(BeNil())

		list, err := dir.Readdir(-1)
		Expect(err).To(BeNil())

		Expect(list).To(HaveLen(1))
		Expect(list[0].Name()).To(Equal("test.txt"))
	})

	It("reads merged dirs", func() {
		_f(baseName, "1111.txt")
		_f(overlayName, "2222.json")

		dir, err := mergedDir.Open(".")
		Expect(err).To(BeNil())

		list, err := dir.Readdir(-1)
		Expect(err).To(BeNil())

		Expect(list).To(HaveLen(2))
		Expect(list[0].Name()).To(Equal("1111.txt"))
		Expect(list[1].Name()).To(Equal("2222.json"))
	})
})

func _f(dir, name string, content ...string) string {
	path := filepath.Join(dir, name)
	file, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	if len(content) > 0 {
		_, _ = file.WriteString(content[0])
	}
	_ = file.Close()
	return path
}
