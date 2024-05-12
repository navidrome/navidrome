package merge_test

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/navidrome/navidrome/utils/merge"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMergeFS(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MergeFS Suite")
}

var _ = Describe("FS", func() {
	var baseName, overlayName string
	var mergedDir fs.FS

	BeforeEach(func() {
		baseName, _ = os.MkdirTemp("", "merge_fs_base_test")
		overlayName, _ = os.MkdirTemp("", "merge_fs_overlay_test")
		baseDir := os.DirFS(baseName)
		overlayDir := os.DirFS(overlayName)
		mergedDir = merge.FS{Base: baseDir, Overlay: overlayDir}
	})
	AfterEach(func() {
		_ = os.RemoveAll(baseName)
		_ = os.RemoveAll(overlayName)
	})

	It("reads from Base dir if not found in Overlay", func() {
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

		content, err := io.ReadAll(file)
		Expect(err).To(BeNil())
		Expect(string(content)).To(Equal("overridden"))
	})

	It("reads only files from Base if Overlay is empty", func() {
		_f(baseName, "test.txt")

		dir, err := mergedDir.Open(".")
		Expect(err).To(BeNil())

		list, err := dir.(fs.ReadDirFile).ReadDir(-1)
		Expect(err).To(BeNil())

		Expect(list).To(HaveLen(1))
		Expect(list[0].Name()).To(Equal("test.txt"))
	})

	It("reads merged dirs", func() {
		_f(baseName, "1111.txt")
		_f(overlayName, "2222.json")

		dir, err := mergedDir.Open(".")
		Expect(err).To(BeNil())

		list, err := dir.(fs.ReadDirFile).ReadDir(-1)
		Expect(err).To(BeNil())

		Expect(list).To(HaveLen(2))
		Expect(list[0].Name()).To(Equal("1111.txt"))
		Expect(list[1].Name()).To(Equal("2222.json"))
	})

	It("allows to seek to the beginning of the directory", func() {
		_f(baseName, "1111.txt")
		_f(baseName, "2222.txt")
		_f(baseName, "3333.txt")

		dir, err := mergedDir.Open(".")
		Expect(err).To(BeNil())

		list, _ := dir.(fs.ReadDirFile).ReadDir(2)
		Expect(list).To(HaveLen(2))
		Expect(list[0].Name()).To(Equal("1111.txt"))
		Expect(list[1].Name()).To(Equal("2222.txt"))

		list, _ = dir.(fs.ReadDirFile).ReadDir(2)
		Expect(list).To(HaveLen(1))
		Expect(list[0].Name()).To(Equal("3333.txt"))
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
