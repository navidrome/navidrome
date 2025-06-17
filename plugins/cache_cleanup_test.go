package plugins

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("purgeCacheBySize", func() {
	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "cache_test")
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(os.RemoveAll, tmpDir)
	})

	It("removes oldest entries when above the size limit", func() {
		oldDir := filepath.Join(tmpDir, "d1")
		newDir := filepath.Join(tmpDir, "d2")
		Expect(os.Mkdir(oldDir, 0700)).To(Succeed())
		Expect(os.Mkdir(newDir, 0700)).To(Succeed())

		oldFile := filepath.Join(oldDir, "old")
		newFile := filepath.Join(newDir, "new")
		Expect(os.WriteFile(oldFile, []byte("xx"), 0600)).To(Succeed())
		Expect(os.WriteFile(newFile, []byte("xx"), 0600)).To(Succeed())

		oldTime := time.Now().Add(-2 * time.Hour)
		Expect(os.Chtimes(oldFile, oldTime, oldTime)).To(Succeed())

		purgeCacheBySize(tmpDir, "3")

		_, err := os.Stat(oldFile)
		Expect(os.IsNotExist(err)).To(BeTrue())
		_, err = os.Stat(oldDir)
		Expect(os.IsNotExist(err)).To(BeTrue())

		_, err = os.Stat(newFile)
		Expect(err).ToNot(HaveOccurred())
	})

	It("does nothing when below the size limit", func() {
		dir1 := filepath.Join(tmpDir, "a")
		dir2 := filepath.Join(tmpDir, "b")
		Expect(os.Mkdir(dir1, 0700)).To(Succeed())
		Expect(os.Mkdir(dir2, 0700)).To(Succeed())

		file1 := filepath.Join(dir1, "f1")
		file2 := filepath.Join(dir2, "f2")
		Expect(os.WriteFile(file1, []byte("x"), 0600)).To(Succeed())
		Expect(os.WriteFile(file2, []byte("x"), 0600)).To(Succeed())

		purgeCacheBySize(tmpDir, "10MB")

		_, err := os.Stat(file1)
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Stat(file2)
		Expect(err).ToNot(HaveOccurred())
	})
})
